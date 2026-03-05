package start_session

import (
	"fmt"
	"log"
	"os"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/cloud/alibaba_cloud"
	"github.com/aliyunidaas/alibaba-cloud-idaas/session_manager"
	"github.com/aliyunidaas/alibaba-cloud-idaas/utils"
	"github.com/gorilla/websocket"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

var (
	stringFlagConfig = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "IDaaS Config",
	}
	stringFlagProfile = &cli.StringFlag{
		Name:    "profile",
		Aliases: []string{"p"},
		Usage:   "IDaaS Profile",
	}

	boolFlagForceNew = &cli.BoolFlag{
		Name:    "force-new",
		Aliases: []string{"N"},
		Usage:   "Force fetch cloud token, ignore all cache",
	}
	boolFlagForceNewCloudToken = &cli.BoolFlag{
		Name:  "force-new-cloud-token",
		Usage: "Force fetch cloud token (lower cache enabled)",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		stringFlagConfig,
		stringFlagProfile,
		boolFlagForceNew,
		boolFlagForceNewCloudToken,
	}
	return &cli.Command{
		Name:  "start-session",
		Usage: "Start Alibaba Cloud ECS session manager",
		Flags: flags,
		Action: func(context *cli.Context) error {
			configFilename := context.String("config")
			profile := context.String("profile")
			forceNew := context.Bool("force-new")
			forceNewCloudToken := context.Bool("force-new-cloud-token")
			return startSessionManager(configFilename, profile, forceNew, forceNewCloudToken)
		},
	}
}

func startSessionManager(configFilename, profile string, forceNew, forceNewCloudToken bool) error {
	options := &cloud.FetchCloudStsOptions{
		ForceNew:           forceNew,
		ForceNewCloudToken: forceNewCloudToken,
	}

	sts, _, err := cloud.FetchCloudStsFromDefaultConfig(configFilename, profile, options)
	if err != nil {
		return err
	}

	alibabaCloudSts, ok := sts.(*alibaba_cloud.StsToken)
	if !ok {
		return fmt.Errorf("allows Alibaba cloud STS token only")
	}
	sessionManagerWebSocketUrl, err := startSession(alibabaCloudSts)
	if err != nil {
		return err
	}
	connection, _, err := websocket.DefaultDialer.Dial(*sessionManagerWebSocketUrl, nil)
	if err != nil {
		return err
	}
	defer connection.Close()

	done := make(chan struct{})
	inputDone := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, data, err := connection.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}
			//fmt.Printf("Data type: %d\n", dataType)
			message, err := session_manager.DecodeMessage(data)
			if err != nil {
				log.Println("Decode error:", err, message)
				continue
			}
			switch message.MsgType {
			case session_manager.Output:
				_, _ = os.Stdout.Write(message.Payload)
				break
			}
			//fmt.Printf("Received from server: %v\n", message)
		}
	}()

	go func() {
		defer close(inputDone)
		//scanner := bufio.NewScanner(os.Stdin)
		//for {
		//	fmt.Print("Enter message to send: ")
		//	if !scanner.Scan() {
		//		break // 用户输入结束（例如 Ctrl+D）
		//	}
		//	text := scanner.Text()
		//
		//	// 发送文本消息
		//	err := connection.WriteMessage(websocket.TextMessage, []byte(text))
		//	if err != nil {
		//		log.Println("Write error:", err)
		//		return
		//	}
		//}
		fd := int(os.Stdin.Fd())
		if !term.IsTerminal(fd) {
			log.Fatal("Standard input is not a terminal.")
		}

		// Read the initial state of the terminal so we can restore it later.
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			log.Fatalf("Failed to set raw mode: %v", err)
		}
		defer func() { // Restore the original terminal state when the program exits
			if restoreErr := term.Restore(fd, oldState); restoreErr != nil {
				log.Printf("Error restoring terminal state: %v", restoreErr)
			} else {
				fmt.Println("\nTerminal state restored. Goodbye!")
			}
		}()

		buf := make([]byte, 3) // Buffer to hold the key press. 3 bytes are enough for most keys and escape sequences.

		for {
			n, err := os.Stdin.Read(buf[:cap(buf)]) // Read into the buffer
			if err != nil {
				log.Printf("Error reading input: %v", err)
				break
			}

			// The actual number of bytes read is n
			keyData := buf[:n]

			// Handle the key press based on the byte(s) received
			switch {
			case n == 1 && keyData[0] == 'q':
				fmt.Print("\n'q' pressed. Quitting...\n")
				return
			case n == 1:
				// Printable ASCII characters or common control characters like \r, \n, \t etc.
				char := rune(keyData[0])
				fmt.Printf("\nPressed character: '%c' (ASCII: %d)\n", char, keyData[0])
			default:
				// This handles multi-byte sequences, like special keys (arrows, function keys).
				// Arrow keys typically send an escape sequence starting with 27 (ESC).
				if keyData[0] == 27 { // ESC character
					if n == 3 && keyData[1] == 91 { // '[' character
						switch keyData[2] {
						case 65:
							fmt.Print("\nUp arrow pressed\n")
						case 66:
							fmt.Print("\nDown arrow pressed\n")
						case 67:
							fmt.Print("\nRight arrow pressed\n")
						case 68:
							fmt.Print("\nLeft arrow pressed\n")
						default:
							// Could be other escape sequences, print as hex for debugging
							fmt.Printf("\nUnknown escape sequence: % x\n", keyData)
						}
					} else {
						// Could be just ESC or another type of escape sequence
						fmt.Printf("\nEscape sequence or ESC key pressed: % x\n", keyData)
					}
				} else {
					// Some other multi-byte sequence
					fmt.Printf("\nMulti-byte key pressed (hex): % x\n", keyData)
				}
			}
		}
	}()

	select {
	case <-done:
		log.Println("Server closed the connection.")
	case <-inputDone:
		log.Println("User closed the client.")
	}

	return nil
}

func startSession(alibabaCloudSts *alibaba_cloud.StsToken) (*string, error) {
	client, err := createClient(alibabaCloudSts)
	if err != nil {
		return nil, err
	}
	startTerminalSessionRequest := &ecs20140526.StartTerminalSessionRequest{
		RegionId:   tea.String("cn-hangzhou"),
		InstanceId: []*string{tea.String("i-bp1gmepco3y79bwglz82")},
	}
	runtime := &util.RuntimeOptions{}
	startTerminalSessionResponse, err := client.StartTerminalSessionWithOptions(startTerminalSessionRequest, runtime)
	if err != nil {
		return nil, err
	}
	utils.Stderr.Println(fmt.Sprintf("SessionID    : %s", *startTerminalSessionResponse.Body.SessionId))
	utils.Stderr.Println(fmt.Sprintf("WebSocket URL: %s", *startTerminalSessionResponse.Body.WebSocketUrl))
	return startTerminalSessionResponse.Body.WebSocketUrl, nil
}

func createClient(alibabaCloudSts *alibaba_cloud.StsToken) (*ecs20140526.Client, error) {
	cred, err := credential.NewCredential(&credential.Config{
		Type:            tea.String("sts"),
		AccessKeyId:     tea.String(alibabaCloudSts.AccessKeyId),
		AccessKeySecret: tea.String(alibabaCloudSts.AccessKeySecret),
		SecurityToken:   tea.String(alibabaCloudSts.StsToken),
	})
	if err != nil {
		return nil, err
	}

	config := &openapi.Config{
		Credential: cred,
	}
	// Endpoint 请参考 https://api.aliyun.com/product/Ecs
	config.Endpoint = tea.String("ecs.cn-hangzhou.aliyuncs.com") // TODO
	client := &ecs20140526.Client{}
	client, err = ecs20140526.NewClient(config)
	return client, err
}
