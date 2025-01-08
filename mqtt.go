package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	exClient mqtt.Client
	inClient mqtt.Client
	exChoke  chan [2]string
	inChoke  chan [2]string
}

type MqttMessage struct {
	Topic   string
	Payload string
}

const (
	MQTT_EXTERNAL_BROKER_URI = "tcp://127.0.0.1:1883"
	MQTT_INTERNAL_BROKER_URI = "tcp://127.0.0.1:1883"
	MQTT_EXTERNAL_CLIENT_ID  = "MqttEx-grpcd"
	MQTT_INTERNAL_CLIENT_ID  = "MqttIn-grpcd"

	//Info
	MQTT_TOPIC_INFO = "info"
	//Status
	MQTT_TOPIC_STATUS_IO        = "status/io/#"
	MQTT_TOPIC_STATUS_NETWORK   = "status/network"
	MQTT_TOPIC_STATUS_ALPR      = "status/alpr"
	MQTT_TOPIC_STATUS_VIDEO     = "status/video/#"
	MQTT_TOPIC_STATUS_SENSOR    = "status/sensor/lux"
	MQTT_TOPIC_STATUS_WATERMARK = "status/watermark/#"
	MQTT_TOPIC_STATUS_FOTA      = "status/fota/percentage"
	//Factory
	MQTT_TOPIC_FACTORY = "factory/info/#"
	//Configure
	MQTT_TOPIC_CONFIG_IO        = "config/io/#"
	MQTT_TOPIC_CONFIG_SYSTEM    = "config/system/#"
	MQTT_TOPIC_CONFIG_NETWROK   = "config/network/#"
	MQTT_TOPIC_CONFIG_VIDEO     = "config/video/#"
	MQTT_TOPIC_CONFIG_WATERMARK = "config/watermark/#"
	//global/factoryMode/TestPkg
	MQTT_TOPIC_CONFIG_TDB = "global/factoryMode/TestPkg"
	//facotry/
	MQTT_TOPIC_CONFIG_FACTORY = "factory/#"
	//Update
	MQTT_TOPIC_UPDATE_TIMEZONE = "update/timezone"
)

var mqttLock sync.Mutex
var MqttClient *Client
var exChoke = make(chan [2]string)
var inChoke = make(chan [2]string)
var MqttPublishChannel = make(chan MqttMessage, 100) // Buffered channel
var chanNum = 2

func MqttNewClient() *Client {
	c := &Client{}
	//c.exClient = createExClient(MQTT_EXTERNAL_BROKER_URI, MQTT_EXTERNAL_CLIENT_ID)
	c.inClient = createInClient(MQTT_INTERNAL_BROKER_URI, MQTT_INTERNAL_CLIENT_ID)
	return c
}

func (m *Client) Publish(clientId, topic, data string) mqtt.Token {
	if clientId == MQTT_EXTERNAL_CLIENT_ID {
		//Log.Infof("%s->%s(VHA-10)", MQTT_EXTERNAL_CLIENT_ID, data)
		return m.exClient.Publish(topic, 0, false, data)
	} else if clientId == MQTT_INTERNAL_CLIENT_ID {
		//Log.Infof("%s->%s", MQTT_INTERNAL_CLIENT_ID, data)
		Log.Infof("\nMqtt Topic:[%s], MSG=[%s]", topic, data)
		return m.inClient.Publish(topic, 0, false, data)
	} else {
		Log.Println("Not identify Mqtt client")
		return nil
	}
}

func createExClient(brokerIp, id string) mqtt.Client {
	opts := mqtt.NewClientOptions().AddBroker(brokerIp).SetClientID(id)
	opts.SetDefaultPublishHandler(monitorEx)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}

	if token := client.Subscribe(MQTT_TOPIC_INFO, 0, mqttExHandler); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}

	return client
}

func createInClient(brokerIp, id string) mqtt.Client {
	opts := mqtt.NewClientOptions().AddBroker(brokerIp).SetClientID(id)
	opts.SetDefaultPublishHandler(monitorIn)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}

	//Info
	if token := client.Subscribe(MQTT_TOPIC_INFO, 0, mqttIn_InfoHandler); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}

	//Status
	if token := client.Subscribe(MQTT_TOPIC_STATUS_IO, 0, mqttIn_Status_IO_Handler); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}
	if token := client.Subscribe(MQTT_TOPIC_STATUS_NETWORK, 0, mqttIn_Status_Network_Handler); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}
	if token := client.Subscribe(MQTT_TOPIC_STATUS_VIDEO, 0, mqttIn_Status_Video_Handler); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}
	if token := client.Subscribe(MQTT_TOPIC_STATUS_SENSOR, 0, mqttIn_Status_Sensor_Handler); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}
	if token := client.Subscribe(MQTT_TOPIC_STATUS_WATERMARK, 0, mqttIn_Status_Watermark_Handler); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}
	if token := client.Subscribe(MQTT_TOPIC_STATUS_ALPR, 0, mqttIn_Status_AlprHandler); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}
	if token := client.Subscribe(MQTT_TOPIC_UPDATE_TIMEZONE, 0, mqttIn_Update_Timezone_Handler); token.Wait() && token.Error() != nil {
		Log.Println(token.Error())
	}

	Log.Println(" [>>>Mqtt Init Done>>>]")

	return client
}

func monitorEx(c mqtt.Client, msg mqtt.Message) {
	exChoke <- [2]string{msg.Topic(), string(msg.Payload())}
}

func monitorIn(c mqtt.Client, msg mqtt.Message) {
	inChoke <- [2]string{msg.Topic(), string(msg.Payload())}
}

func mqttIn_InfoHandler(client mqtt.Client, msg mqtt.Message) {
	Log.Debugf("Recv topic: %s, data: %s", msg.Topic(), msg.Payload())
	json.Unmarshal([]byte(msg.Payload()), &AppConfig.System)

	Log.Infof("Firmware Version: %s\n", AppConfig.System.FWVersion)
	Log.Infof("Time: %s\n", AppConfig.System.Time)
	Log.Infof("Serial Number: %s\n", AppConfig.System.SerialNo)
	Log.Infof("SKU Name: %s\n", AppConfig.System.SKUName)
	Log.Infof("Device Name: %s\n", AppConfig.System.DeviceName)
	Log.Infof("MAC: %s\n", AppConfig.System.MAC)
	Log.Infof("AlprEnabled: %v\n", AppConfig.System.AlprEnabled)
}

func mqttIn_Status_AlprHandler(client mqtt.Client, msg mqtt.Message) {
	Log.Debugf("Recv topic: %s, data: %s", msg.Topic(), msg.Payload())
	json.Unmarshal([]byte(msg.Payload()), &AppConfig.System)
	Log.Infof("Alpr: %v\n", AppConfig.System.AlprEnabled)
}

func mqttIn_Status_Video_Handler(client mqtt.Client, msg mqtt.Message) {
	//{Resolution: "1920x1080",StreamFormat: "H.264",BitRate: 5000,Type: "VBR"} [0/1]
	Log.Debugf("Recv topic: %s, data: %s", msg.Topic(), msg.Payload())
	topic := msg.Topic()

	parts := strings.Split(topic, "/")

	key := parts[2]
	num, err := strconv.Atoi(key)
	if err != nil {
		Log.Println("Error converting key to int:", err)
		return
	}
	if num < chanNum {
		if AppConfig.Videos == nil {
			AppConfig.Videos = make(map[string]VideoConfig)
		}
		videoConfig, exists := AppConfig.Videos[key]
		if !exists {
			videoConfig = VideoConfig{}
			Log.Println("Assign VideoConfig{}")
		}
		err := json.Unmarshal([]byte(msg.Payload()), &videoConfig)
		if err != nil {
			Log.Infof("Error unmarshaling JSON:%v", err)
		}
		AppConfig.Videos[key] = videoConfig
		Log.Println(AppConfig.Videos[key])
	} else {
		Log.Printf("Invalid Channel")
	}
}

func mqttIn_Status_IO_Handler(client mqtt.Client, msg mqtt.Message) {
	Log.Debugf("Recv topic: %s, data: %s", msg.Topic(), msg.Payload())
	topic := msg.Topic()
	parts := strings.Split(topic, "/")

	key := parts[3]
	num, err := strconv.Atoi(key)
	if err != nil {
		Log.Println("Error converting key to int:", err)
		return
	}
	switch parts[1] {
	case "io":
		if parts[2] == "led" && num < chanNum {
			if AppConfig.LEDs == nil {
				AppConfig.LEDs = make(map[string]LEDConfig)
			}
			ledConfig, exists := AppConfig.LEDs[key]
			if !exists {
				ledConfig = LEDConfig{}
				Log.Println("Assign LEDConfig{}")
			}
			err := json.Unmarshal([]byte(msg.Payload()), &ledConfig)
			if err != nil {
				Log.Println("Error unmarshaling JSON:", err)
			}
			AppConfig.LEDs[key] = ledConfig
			//Log.Infof("[%s] StatusLed: %s\n", key, AppConfig.LEDs[key].StatusLed)
			//Log.Infof("[%s] RecLedOn: %v\n", key, AppConfig.LEDs[key].RecLedOn)
		} else {
			Log.Infof("Invaild arguments, part=%s, num=%d\n", parts[2], num)
		}
	default:
		Log.Infof("Not Identify node:%s", parts[1])
	}
}

func mqttIn_Status_Network_Handler(client mqtt.Client, msg mqtt.Message) {
	Log.Debugf("Recv topic: %s, data: %s", msg.Topic(), msg.Payload())
	err := json.Unmarshal([]byte(msg.Payload()), &AppConfig.Network)
	if err != nil {
		Log.Println("Error unmarshaling JSON:", err)
	}

	Log.Infof("Network:IPv4, IP=%s", AppConfig.Network.IPv4)
	Log.Infof("Network:IPv6, IP=%s", AppConfig.Network.IPv6)
}

func mqttIn_Status_Sensor_Handler(client mqtt.Client, msg mqtt.Message) {
	//{Mode: "Day/Night", Lux: <value>}
	Log.Debugf("Recv topic: %s, data: %s", msg.Topic(), msg.Payload())

	err := json.Unmarshal([]byte(msg.Payload()), &AppConfig.DayNightMode)
	if err != nil {
		Log.Println("Error unmarshaling JSON:", err)
	}

	Log.Infof("Mode=%s, Lux=%d", AppConfig.DayNightMode.Mode, AppConfig.DayNightMode.Lux)
}

func mqttIn_Status_Watermark_Handler(client mqtt.Client, msg mqtt.Message) {
	//{ Username:"DefaultUser",
	//  OptionUserName:   true,
	//  OptionDeviceName: true,
	//  OptionGPS:        true,
	//  OptionTime:       true,
	//  OptionLogo:       true} [0/1]
	Log.Debugf("Recv topic: %s, data: %s", msg.Topic(), msg.Payload())
	topic := msg.Topic()

	parts := strings.Split(topic, "/")

	key := parts[2]
	num, err := strconv.Atoi(key)
	if err != nil {
		Log.Println("Error converting key to int:", err)
		return
	}
	if num < chanNum {
		if AppConfig.Watermarks == nil {
			AppConfig.Watermarks = make(map[string]WatermarkConfig)
		}
		osdConfig, exists := AppConfig.Watermarks[key]
		if !exists {
			osdConfig = WatermarkConfig{}
			Log.Println("Assign WatermarkConfig{}")
		}
		err := json.Unmarshal([]byte(msg.Payload()), &osdConfig)
		if err != nil {
			Log.Infof("Error unmarshaling JSON:%v", err)
		}
		AppConfig.Watermarks[key] = osdConfig
		Log.Println(AppConfig.Watermarks[key])
	} else {
		Log.Printf("Invalid Channel")
	}
}

func mqttIn_Update_Timezone_Handler(client mqtt.Client, msg mqtt.Message) {
	// {"TzStr":"UTC-08:00"}
	Log.Debugf("Recv topic: %s, data: %s", msg.Topic(), msg.Payload())

	var data struct {
		TzStr string `json:"TzStr"`
	}
	err := json.Unmarshal(msg.Payload(), &data)
	if err != nil {
		Log.Println("Error unmarshaling JSON:", err)
		return
	}

	os.Setenv("TZ", data.TzStr)
	Log.Infof("Timezone set to: %s", data.TzStr)
}

func mqttExHandler(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	value := string(msg.Payload())
	Log.Infof("Topic:%s, MSG:%s\n", topic, value)
}

func StartMqttInLoop() {
	Log.Infof("Run")
	for {
		incoming := <-inChoke
		Log.Infof("Recv:Topic:%s, Msg:%s\n", incoming[0], incoming[1])
	}
}

func StartMqttExLoop() {
	Log.Println("MqttEx Monitor start")
	for {
		incoming := <-exChoke
		Log.Infof("Recv:Topic:%s, Msg:%s\n", incoming[0], incoming[1])
	}
}

func MqttInit() {
	MqttClient = MqttNewClient()
}

func StartMqttWorker() {
	go func() {
		for message := range MqttPublishChannel {
			mqttLock.Lock()
			token := MqttClient.Publish(MQTT_INTERNAL_CLIENT_ID, message.Topic, message.Payload)
			token.Wait()
			if token.Error() != nil {
				Log.Errorf("Failed to publish to topic %s: %v", message.Topic, token.Error())
			}
			mqttLock.Unlock()
		}
		Log.Info("Mqtt worker stopped")
	}()
}
