package serve

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var (
	startup = time.Now().UnixMilli()
)

var (
	intFlagPort = &cli.IntFlag{
		Name:    "port",
		Aliases: []string{"p"},
		Usage:   "Port (default 1127)",
	}
	stringFlagUnsafeListenHost = &cli.StringFlag{
		Name:  "unsafe-listen-host",
		Usage: "Default listen 127.0.0.1, use this flag can assign to 0.0.0.0",
	}
	stringFlagSsrfToken = &cli.StringFlag{
		Name:  "ssrf-token",
		Usage: "SSRF Token, send in query header X-Aliyun-Parameters-Secrets-Token",
	}
	boolFlagUnsafeDisableSsrf = &cli.BoolFlag{
		Name:  "unsafe-disable-ssrf",
		Usage: "Disable SSRF feature",
	}
)

func BuildCommand() *cli.Command {
	flags := []cli.Flag{
		intFlagPort,
		stringFlagUnsafeListenHost,
		stringFlagSsrfToken,
		boolFlagUnsafeDisableSsrf,
	}
	return &cli.Command{
		Name:  "serve",
		Usage: "Serve local server",
		Flags: flags,
		Action: func(context *cli.Context) error {
			ssrfToken := context.String("ssrf-token")
			unsafeDisableSsrf := context.Bool("unsafe-disable-ssrf")
			if ssrfToken == "" {
				if !unsafeDisableSsrf {
					return errors.New("SSRF token is required, unless --unsafe-disable-ssrf is set")
				}
			}

			unsafeListenHost := context.String("unsafe-listen-host")
			port := context.Int("port")
			if port == 0 {
				port = 1127
			}
			if port <= 0 || port > 65535 {
				return fmt.Errorf("invalid port %d", port)
			}
			listenHostAndPort := getListenHostAndPort(unsafeListenHost, port)
			return serve(listenHostAndPort, &HttpServeOptions{
				SsrfToken: ssrfToken,
			})
		},
	}
}

func serve(listenHostAndPort string, serveOptions *HttpServeOptions) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRoot(w, r, serveOptions)
	})
	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		handleVersion(w, r, serveOptions)
	})
	http.HandleFunc("/cloud_token", func(w http.ResponseWriter, r *http.Request) {
		handleCloudToken(w, r, serveOptions)
	})

	fmt.Printf("Listen at %s...", listenHostAndPort)
	return http.ListenAndServe(listenHostAndPort, nil)
}

func handleRoot(w http.ResponseWriter, r *http.Request, serveOptions *HttpServeOptions) {
	if !isRequestAllowed(w, r, serveOptions) {
		return
	}
	printResponse(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "Resource not found.",
	})
}

func handleVersion(w http.ResponseWriter, r *http.Request, serveOptions *HttpServeOptions) {
	if !isRequestAllowed(w, r, serveOptions) {
		return
	}
	printResponse(w, http.StatusOK, VersionResponse{
		Name:    "alibaba-cloud-idaas",
		Version: constants.AlibabaCloudIdaasCliVersion,
		Startup: startup,
	})
}

func getListenHostAndPort(unsafeListenHost string, port int) string {
	var listenHostAndPort string
	if unsafeListenHost == "" {
		listenHostAndPort = fmt.Sprintf("127.0.0.1:%d", port)
	} else {
		// may be unsafe
		listenHostAndPort = fmt.Sprintf("%s:%d", unsafeListenHost, port)
	}
	return listenHostAndPort
}
