package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"

	log "github.com/sirupsen/logrus"
	goauth "golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/p000ic/go-fosite-mongo/examples/authorizationserver"
	"github.com/p000ic/go-fosite-mongo/examples/oauth2client"
	"github.com/p000ic/go-fosite-mongo/examples/resourceserver"
)

// A valid oauth2 client (check the store) that additionally requests an OpenID Connect id token
var clientConf = goauth.Config{
	ClientID:     "my-client",
	ClientSecret: "foobar",
	RedirectURL:  "http://localhost:3846/callback",
	Scopes:       []string{"photos", "openid", "offline"},
	Endpoint: goauth.Endpoint{
		TokenURL: "http://localhost:3846/oauth2/token",
		AuthURL:  "http://localhost:3846/oauth2/auth",
	},
}

// The same thing (valid oauth2 client) but for using the client credentials grant
var appClientConf = clientcredentials.Config{
	ClientID:     "my-client",
	ClientSecret: "foobar",
	Scopes:       []string{"fosite"},
	TokenURL:     "http://localhost:3846/oauth2/token",
}

func main() {
	// configure HTTP server.
	port := "8000"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	srv := &http.Server{Addr: ":" + port}
	fmt.Printf("server starting at port %s\n", port)
	// ### oauth2 server ###
	authorizationserver.RegisterHandlers() // the authorization server (fosite)
	// ### oauth2 client ###
	http.HandleFunc("/", oauth2client.HomeHandler(clientConf)) // show some links on the index
	// the following handlers are oauth2 consumers
	http.HandleFunc("/client", oauth2client.ClientEndpoint(appClientConf)) // complete a client credentials flow
	http.HandleFunc("/owner", oauth2client.OwnerHandler(clientConf))       // complete a resource owner password credentials flow
	http.HandleFunc("/callback", oauth2client.CallbackHandler(clientConf)) // the oauth2 callback endpoint
	// ### protected resource ###
	http.HandleFunc("/protected", resourceserver.ProtectedEndpoint(appClientConf))
	fmt.Println("Please open your web browser at http://localhost:" + port)
	_ = exec.Command("open", "http://localhost:"+port).Run()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			// unexpected error
			log.WithError(err).Error("error starting http server!")
		}
	}()
	log.Infof("server started at port %s", port)

	// Set up signal capturing to know when the server is being killed..
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Wait for SIGINT (pkill -2)
	<-stop

	// Gracefully shutdown the HTTP server.
	log.Info("shutting down server...")
	if err := srv.Shutdown(context.TODO()); err != nil {
		// failure/timeout shutting down the server gracefully
		log.WithError(err).Error("error gracefully shutting down http server")
	}

	// wait for graceful shutdown.
	wg.Wait()
	log.Error("server stopped!")
}
