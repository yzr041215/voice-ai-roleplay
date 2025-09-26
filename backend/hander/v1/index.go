package V1

const index = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>VAD WebSocket Demo</title>
  <style>
    body { font-family: sans-serif; padding: 20px; }
    #status { margin-top: 20px; font-size: 18px; color: green; }
  </style>
</head>
<body>
  <h1>VAD WebSocket Demo</h1>
  <button id="startBtn">🎙️ 开启录音</button>
  <button id="stopBtn">⏹️ 停止录音</button>
  <div id="status">等待中...</div>

  <script>
    let ws;
    let audioContext;
    let processor;
    let source;
    let stream;

    const FRAME_SIZE = 640; // 每帧字节数 (20ms @ 16kHz, PCM16)
    let pcmBuffer = new Uint8Array(0); // 缓冲区

    const startBtn = document.getElementById("startBtn");
    const stopBtn = document.getElementById("stopBtn");
    const statusDiv = document.getElementById("status");

    // 播放队列
    let audioQueue = [];
    let audioPlaying = false;

    startBtn.onclick = async () => {
      ws = new WebSocket("wss://204.141.218.207:8080/v1/ws");
      ws.binaryType = "arraybuffer";

      ws.onopen = () => {
        console.log("WebSocket connected");
      };

      ws.onmessage = (event) => {
        if (typeof event.data === "string") {
          try {
            const msg = JSON.parse(event.data);
            if (msg.isVad !== undefined) {
              statusDiv.textContent = msg.isVad ? "🎤 正在说话..." : "🤐 静音";
            }
          } catch (e) {
            console.log("message:", event.data);
          }
        } else if (event.data instanceof ArrayBuffer) {
          // 收到 TTS PCM 数据，放入播放队列
          audioQueue.push(event.data);
          playAudioQueue();
        }
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
        const pcm16 = floatTo16BitPCM(input); // Int16Array

        // 拼接缓冲区
        let tmp = new Int16Array(pcmBuffer.length + pcm16.length);
        tmp.set(pcmBuffer, 0);
        tmp.set(pcm16, pcmBuffer.length);
        pcmBuffer = tmp;

        // 每帧 320 采样点 = 640 字节
        const samplesPerFrame = FRAME_SIZE / 2;
        while (pcmBuffer.length >= samplesPerFrame) {
          const frame = pcmBuffer.slice(0, samplesPerFrame);
          ws.send(frame.buffer); // 发送完整的 PCM16 帧
          pcmBuffer = pcmBuffer.slice(samplesPerFrame);
        }
      };
    };

    stopBtn.onclick = () => {
      if (ws) {
        ws.send("stop");
        ws.close();
      }
      if (processor) processor.disconnect();
      if (source) source.disconnect();
      if (stream) stream.getTracks().forEach(track => track.stop());
      if (audioContext) audioContext.close();
      pcmBuffer = new Uint8Array(0);
      audioQueue = [];
      audioPlaying = false;
      statusDiv.textContent = "已停止";
    };

    function floatTo16BitPCM(float32Array) {
      const pcm16 = new Int16Array(float32Array.length);
      for (let i = 0; i < float32Array.length; i++) {
        let s = Math.max(-1, Math.min(1, float32Array[i]));
        pcm16[i] = s < 0 ? s * 0x8000 : s * 0x7fff;
      }
      return pcm16;
    }

    function playAudioQueue() {
      if (audioPlaying || audioQueue.length === 0) return;
      audioPlaying = true;

      const buffer = audioQueue.shift();
      const audioBuffer = audioContext.createBuffer(1, buffer.byteLength / 2, 16000);
      const channelData = audioBuffer.getChannelData(0);
      const view = new DataView(buffer);
      for (let i = 0; i < channelData.length; i++) {
        const sample = view.getInt16(i * 2, true); // little-endian
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
