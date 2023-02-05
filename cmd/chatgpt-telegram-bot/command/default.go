package command

import (
	"flag"
	"github.com/pefish/go-commander"
	go_config "github.com/pefish/go-config"
	go_error "github.com/pefish/go-error"
	go_http "github.com/pefish/go-http"
	go_logger "github.com/pefish/go-logger"
	telegram_robot "github.com/pefish/telegram-bot-manager/pkg/telegram-robot"
	"time"
)

type DefaultCommand struct {
	robot *telegram_robot.Robot
}

func NewDefaultCommand() *DefaultCommand {
	return &DefaultCommand{

	}
}

func (dc *DefaultCommand) DecorateFlagSet(flagSet *flag.FlagSet) error {
	return nil
}

func (dc *DefaultCommand) OnExited(data *commander.StartData) error {
	if dc.robot != nil {
		err := dc.robot.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (dc *DefaultCommand) Start(data *commander.StartData) error {
	tgToken, err := go_config.ConfigManagerInstance.GetString("tg-token")
	if err != nil {
		return go_error.WithStack(err)
	}

	chatgptToken, err := go_config.ConfigManagerInstance.GetString("chatgpt-token")
	if err != nil {
		return go_error.WithStack(err)
	}

	fetchInterval, err := go_config.ConfigManagerInstance.GetUint64("fetch-interval")
	if err != nil {
		return go_error.WithStack(err)
	}

	dc.robot = telegram_robot.NewRobot(tgToken, time.Duration(fetchInterval) * time.Second)
	dc.robot.SetLogger(go_logger.Logger)

	err = dc.robot.Start(data.ExitCancelCtx, data.DataDir, func(command string, data string) string {
		go_logger.Logger.InfoF("命令：%s，问题：%s", command, data)
		if data == "" || command == "" {
			return ""
		}
		var result struct{
			Choices []struct{
				Text string `json:"text"`
			} `json:"choices"`
		}
		_, err := go_http.NewHttpRequester(go_http.WithLogger(go_logger.Logger), go_http.WithTimeout(5 * time.Minute)).PostForStruct(go_http.RequestParam{
			Url:       "https://api.openai.com/v1/completions",
			Params:    map[string]interface{}{
				"model": "text-davinci-003",
				"prompt": data,
				"temperature": 0,
				"max_tokens": 2048,
			},
			Headers: map[string]interface{}{
				"Authorization": "Bearer " + chatgptToken,
			},
		}, &result)
		if err != nil || result.Choices == nil || len(result.Choices) == 0 {
			go_logger.Logger.ErrorF("err: %#v, result: %#v", err, result)
			return "Ops, 出错了，快去联系 pefish 哥哥吧！！！"
		}
		return result.Choices[0].Text
	})
	if err != nil {
		return go_error.WithStack(err)
	}

	<- data.ExitCancelCtx.Done()
	return nil
}

