package domain

type AsrResponse struct {
	Data struct {
		Result struct {
			Additions map[string]string `json:"additions"`
			Text      string            `json:"text"`
		} `json:"result"`
	} `json:"data"`
}

/*
{
  "reqid": "bdf5e1b1bcaca22c7a9248aba2804912",
  "operation": "asr",
  "data": {
    "audio_info": { "duration": 9336 },
    "result": {
      "additions": { "duration": "9336" },
      "text": "七牛的文化是做一个简单的人，做一款简单的产品，做一家简单的公司。"
    }
  }
}
*/
