
curl --location 'https://openai.qiniu.com/v1/voice/tts' \
  --header 'Content-Type: application/json' \
  --header 'Authorization: Bearer sk-9d77b19cf27454ab47ea9b0d9d2db30c7139c5712c0e553bc31eb6fa4dc90bcf' \
  --data '{
  "audio": {
    "voice_type": "zh_male_laobai",
    "encoding": "mp3",
    "speed_ratio": 1.0
  },
  "request": {
    "text": "你好，世界！"
  }
}'


//http://204.141.218.207:9000/mybucket/录音 (2).m4a

curl --location 'https://openai.qiniu.com/v1/voice/asr' \
  --header 'Content-Type: application/json' \
  --header 'Authorization: Bearer sk-9d77b19cf27454ab47ea9b0d9d2db30c7139c5712c0e553bc31eb6fa4dc90bcf' \
  --data '{
  "model": "asr",
  "audio": {
    "format": "mp3",
    "url": "http://204.141.218.207:9000/mybucket/aaa.mp3"
  }
}'
curl --location 'https://openai.qiniu.com/v1/voice/asr' \
  --header 'Content-Type: application/json' \
  --header 'Authorization: Bearer sk-9d77b19cf27454ab47ea9b0d9d2db30c7139c5712c0e553bc31eb6fa4dc90bcf' \
  --data '{
  "model": "asr",
  "audio": {
    "format": "wav",
    "url": "http://204.141.218.207:9000/mybucket/seg_0.wav"
  }
}'

curl --location 'https://openai.qiniu.com/v1/voice/list