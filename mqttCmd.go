package main

import (
	"encoding/json"
)

type Params struct {
	Time struct {
		Time string `json:"time"`
	} `json:"time"`
	Command struct {
		Value string `json:"value"`
	} `json:"command"`
	Log struct {
		Value string `json:"value"`
	} `json:"log"`
	IPv4 struct {
		Value string `json:"IPv4"`
	}
	IPv6 struct {
		Value string `json:"IPv6"`
	}
	AlprStatus struct {
		Value bool `json:"Enable"`
	}

	IO        map[string]LEDConfig       `json:"io"`
	Video     map[string]VideoConfig     `json:"video"`
	Watermark map[string]WatermarkConfig `json:"watermark"`
}

var gMqttCmd Params

func ConfigHandler(target string, ch int32, params interface{}) error {
	tmp, err := json.Marshal(params)
	if err != nil {
		//return Log.Infof("failed to marshal params: %s", err)
		Log.Infof("failed to marshal params: %s", err)
	}

	// Parse JSON into struct
	var jsonObj Params
	if err := json.Unmarshal(tmp, &jsonObj); err != nil {
		//return Log.Infof("failed to unmarshal JSON: %s", err)
		Log.Infof("failed to unmarshal JSON: %s", err)
	}

	// Use the parsed data
	//Log.Debugf("JSON: %+v\n", string(tmp))
	//Log.Debugf("Parsed JSON: %+v\n", jsonObj)

	switch target {
	case "video":
		if ch > 1 {
			Log.Info("Invalid Channel")
		}
		if ch == 0 {
			Log.Infof("Channel %d, [%+v]", ch, jsonObj.Video["0"])
			tmp, err = json.Marshal(jsonObj.Video["0"])
			if err != nil {
				Log.Infof("failed to marshal params: %s", err)
			}
			MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/video/0", string(tmp))
		} else {
			Log.Infof("Channel %d, [%+v]", ch, jsonObj.Video["1"])
			tmp, err = json.Marshal(jsonObj.Video["1"])
			if err != nil {
				Log.Infof("failed to marshal params: %s", err)
			}
			MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/video/1", string(tmp))
		}
	case "io":
		Log.Info("io")
		if ch > 1 {
			Log.Info("Invalid Channel")
		}
		if ch == 0 {
			Log.Infof("Channel %d, [%+v]", ch, jsonObj.IO["0"])
			tmp, err = json.Marshal(jsonObj.IO["0"])
			if err != nil {
				Log.Infof("failed to marshal params: %s", err)
			}
			MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/io/led/0", string(tmp))
		} else {
			Log.Infof("Channel %d, [%+v]", ch, jsonObj.IO["1"])
			tmp, err = json.Marshal(jsonObj.IO["1"])
			if err != nil {
				Log.Infof("failed to marshal params: %s", err)
			}
			MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/io/led/1", string(tmp))
		}
	case "system-setTime":
		Log.Info("system-setTime")
		tmp, err = json.Marshal(jsonObj.Time)
		if err != nil {
			Log.Infof("failed to marshal params: %s", err)
		}
		MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/system/time", string(tmp))
	case "system-command":
		Log.Info("system-command")
		tmp, err = json.Marshal(jsonObj.Command)
		if err != nil {
			Log.Infof("failed to marshal params: %s", err)
		}
		MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/system/command", string(tmp))
	case "system-log":
		Log.Info("system-log")
		MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/system/log", string(jsonObj.Log.Value))
	case "alpr-set":
		tmp, err = json.Marshal(jsonObj.AlprStatus)
		if err != nil {
			Log.Infof("failed to marshal params: %s", err)
		}
		MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/alpr", string(tmp))
	case "network-IPv4":
		Log.Info("network-IPv4")
		tmp, err = json.Marshal(jsonObj.IPv4)
		if err != nil {
			Log.Infof("failed to marshal params: %s", err)
		}
		MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/network", string(tmp))
	case "network-IPv6":
		Log.Info("network-IPv6")
		tmp, err = json.Marshal(jsonObj.IPv6)
		if err != nil {
			Log.Infof("failed to marshal params: %s", err)
		}
		MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/network", string(tmp))
	case "watermark":
		if ch > 1 {
			Log.Info("Invalid Channel")
		}
		if ch == 0 {
			Log.Infof("Channel %d, [%+v]", ch, jsonObj.Watermark["0"])
			tmp, err = json.Marshal(jsonObj.Watermark["0"])
			if err != nil {
				Log.Infof("failed to marshal params: %s", err)
			}
			MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/watermark/0", string(tmp))
		} else {
			Log.Infof("Channel %d, [%+v]", ch, jsonObj.Watermark["1"])
			tmp, err = json.Marshal(jsonObj.Watermark["1"])
			if err != nil {
				Log.Infof("failed to marshal params: %s", err)
			}
			MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, "config/watermark/1", string(tmp))
		}
	}

	return nil
}
