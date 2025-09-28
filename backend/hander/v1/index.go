package V1

const index = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>VAD WebSocket Demo (ASR+LLM+TTS)</title>
  <style>
    body { font-family: sans-serif; padding: 20px; }
    button { margin: 5px; padding: 10px; }
    #status { margin-top: 20px; font-size: 18px; color: green; }
    #asr, #llm { margin-top: 15px; padding: 10px; border: 1px solid #ccc; white-space: pre-wrap; }
  </style>
</head>
<body>
  <h1>VAD WebSocket Demo (ASR+LLM+TTS)</h1>
  <button id="startBtn">🎙️ 开启录音</button>
  <button id="stopBtn">⏹️ 停止录音</button>
  <button id="interruptBtn">⚡ 打断</button>

  <div id="status">等待中...</div>
  <div id="asr"><b>ASR识别:</b> <span id="asrText"></span></div>
  <div id="llm"><b>回复:</b> <span id="llmText"></span></div>

  <script>
    let ws;
    let audioContext;
    let processor;
    let source;
    let stream;

    const FRAME_SIZE = 640; // 20ms @ 16kHz PCM16
    let pcmBuffer = new Int16Array(0);

    // 播放队列（ArrayBuffer 的 bytes）
    let audioQueue = [];
    let audioPlaying = false;

    const startBtn = document.getElementById("startBtn");
    const stopBtn = document.getElementById("stopBtn");
    const interruptBtn = document.getElementById("interruptBtn");
    const statusDiv = document.getElementById("status");
    const asrSpan = document.getElementById("asrText");
    const llmSpan = document.getElementById("llmText");

    startBtn.onclick = async () => {
      ws = new WebSocket("wss://204.141.218.207:8080/v1/ws");
      ws.binaryType = "arraybuffer";

      ws.onopen = () => {
        console.log("WebSocket connected");
        statusDiv.textContent = "已连接，等待音频...";
      };

      ws.onmessage = (event) => {
        // 后端把 domain.Msg JSON 发来
        if (typeof event.data === "string") {
          let msg;
          try {
            msg = JSON.parse(event.data);
          } catch (e) {
            console.warn("非 JSON 文本消息:", event.data);
            return;
          }

          // msg.data 是 base64（因为 server 把 []byte 放到 JSON），或也可能是直接字符串
          const dataField = msg.data;

          // helper: decode base64 to string
          function base64ToString(b64) {
            try {
              // atob -> string bytes (binary string), but if it's UTF-8 JSON we can decode via decodeURIComponent
              const bin = atob(b64);
              // convert binary-string to utf8 string
              let utf8 = "";
              for (let i = 0; i < bin.length; i++) {
                utf8 += String.fromCharCode(bin.charCodeAt(i));
              }
              try {
                return decodeURIComponent(escape(utf8));
              } catch (e) {
                return utf8;
              }
            } catch (e) {
              // not base64? maybe plain JSON string already
              return b64;
            }
          }

          switch (msg.type) {
            case "state": {
              // state.data 是 JSON bytes -> base64
              const inner = base64ToString(dataField);
              try {
                const st = JSON.parse(inner);
                statusDiv.textContent =  '状态: ' + st.state + ' | VAD: ' + (st.isVad ? '🎤 说话中' : '🤐 静音');
              } catch (e) {
                console.warn("解析 state 失败:", e, inner);
              }
              break;
            }
            case "asr_result": {
              const inner = base64ToString(dataField);
              try {
                const asr = JSON.parse(inner);
                asrSpan.textContent = asr.text;
              } catch (e) {
                console.warn("解析 asr_result 失败:", e, inner);
              }
              break;
            }
            case "tts_start": {
              llmSpan.textContent = "";
              break;
            }
            case "tts_chunk": {
              // inner is JSON {"seq":n,"pcm":"<base64>","text": "..."}
              const inner = base64ToString(dataField);
              try {
                const payload = JSON.parse(inner);
                // payload.pcm is base64 of raw pcm bytes (little-endian int16)
                const pcmB64 = payload.pcm;
                const raw = base64ToUint8Array(pcmB64);
                // push ArrayBuffer to queue
                audioQueue.push(raw.buffer);
                playAudioQueue();
                // optionally append LLM text meta
                if (payload.text) {
                  llmSpan.textContent += payload.text;
                }
              } catch (e) {
                console.warn("解析 tts_chunk 失败:", e, inner);
              }
              break;
            }
            case "tts_end": {
              llmSpan.textContent += " ✅";
              break;
            }
            case "intrupt": {
              statusDiv.textContent = "⚡ 已打断";
              break;
            }
            case "translate": {
              const inner = base64ToString(dataField);
              try {
                const tr = JSON.parse(inner);
                llmSpan.textContent = "客户端输入: " + (tr.text || JSON.stringify(tr));
              } catch (e) {
                llmSpan.textContent = "客户端输入: " + inner;
              }
              break;
            }
            default:
              console.log("未知消息:", msg.type, msg);
          }

        } else if (event.data instanceof ArrayBuffer) {
          // 如果服务端也发送了裸 binary（我们当前主要发送 base64 in text），这里依然支持播放裸 PCM
          audioQueue.push(event.data);
          playAudioQueue();
        }
      };

      ws.onclose = () => {
        statusDiv.textContent = "已断开";
      };

      // 获取音频流
      stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      audioContext = new AudioContext({ sampleRate: 16000 });
      source = audioContext.createMediaStreamSource(stream);

      processor = audioContext.createScriptProcessor(1024, 1, 1);
      source.connect(processor);
      processor.connect(audioContext.destination);

      processor.onaudioprocess = (e) => {
        const input = e.inputBuffer.getChannelData(0);
        const pcm16 = floatTo16BitPCM(input);

        // 拼接缓冲区
        let tmp = new Int16Array(pcmBuffer.length + pcm16.length);
        tmp.set(pcmBuffer, 0);
        tmp.set(pcm16, pcmBuffer.length);
        pcmBuffer = tmp;

        const samplesPerFrame = FRAME_SIZE / 2;
        while (pcmBuffer.length >= samplesPerFrame) {
          const frame = pcmBuffer.slice(0, samplesPerFrame);
          // send raw PCM as ArrayBuffer (server AsrStream 接受裸 PCM)
          ws.send(frame.buffer);
          pcmBuffer = pcmBuffer.slice(samplesPerFrame);
        }
      };
    };

    stopBtn.onclick = () => {
      if (ws) {
        ws.close();
        ws = null;
      }
      if (processor) processor.disconnect();
      if (source) source.disconnect();
      if (stream) stream.getTracks().forEach(track => track.stop());
      if (audioContext) audioContext.close();

      pcmBuffer = new Int16Array(0);
      audioQueue = [];
      audioPlaying = false;
      asrSpan.textContent = "";
      llmSpan.textContent = "";
      statusDiv.textContent = "已停止";
    };

    interruptBtn.onclick = () => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        const msg = { type: "intrupt", data: {} };
        ws.send(JSON.stringify(msg));
        console.log("发送打断:", msg);
      }
    };

    function floatTo16BitPCM(float32Array) {
      const pcm16 = new Int16Array(float32Array.length);
      for (let i = 0; i < float32Array.length; i++) {
        let s = Math.max(-1, Math.min(1, float32Array[i]));
        pcm16[i] = s < 0 ? s * 0x8000 : s * 0x7fff;
      }
      return pcm16;
    }

    function base64ToUint8Array(base64) {
      // atob -> binary string -> Uint8Array
      const bin = atob(base64);
      const len = bin.length;
      const bytes = new Uint8Array(len);
      for (let i = 0; i < len; i++) {
        bytes[i] = bin.charCodeAt(i);
      }
      return bytes;
    }

    function playAudioQueue() {
      if (!audioContext) return;
      if (audioPlaying || audioQueue.length === 0) return;
      audioPlaying = true;

      const buffer = audioQueue.shift();
      // buffer is ArrayBuffer of PCM16 little-endian
      const view = new DataView(buffer);
      const samples = buffer.byteLength / 2;
      const audioBuffer = audioContext.createBuffer(1, samples, 16000);
      const channelData = audioBuffer.getChannelData(0);
      for (let i = 0; i < samples; i++) {
        const sample = view.getInt16(i * 2, true);
        channelData[i] = sample / 32768;
      }

      const sourceNode = audioContext.createBufferSource();
      sourceNode.buffer = audioBuffer;
      sourceNode.connect(audioContext.destination);
      sourceNode.start();
      sourceNode.onended = () => {
        audioPlaying = false;
        playAudioQueue();
      };
    }
  </script>
</body>
</html>

`
