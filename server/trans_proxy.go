package server

import (
	"io"
	"net"

	"github.com/jiuzhou-zhao/lumos.git/config"
	"github.com/jiuzhou-zhao/lumos.git/utils"
	"github.com/sirupsen/logrus"
)

type TransProxy struct {
	cfg *config.Config
}

func NewTransProxy(cfg *config.Config) *TransProxy {
	if cfg.EffectMode != config.ModeLocal && cfg.EffectMode != config.ModeRelay {
		logrus.Fatalf("invalid mode: %v", cfg.Mode)
	}

	if cfg.RemoteProxyAddress == "" {
		logrus.Fatal("no remote proxy address")
	}
	return &TransProxy{
		cfg: cfg,
	}
}

func (proxy *TransProxy) Serve() {
	tcpServer := NewTCPServer()

	var clientChan <-chan net.Conn
	var err error
	if proxy.cfg.Secure.TLSEnableFlag.ServerUseTLS {
		clientChan, err = tcpServer.StartTLS(proxy.cfg.ProxyAddress, proxy.cfg.Secure.ServerTLSConfig)
	} else {
		clientChan, err = tcpServer.Start(proxy.cfg.ProxyAddress)
	}

	if err != nil {
		logrus.Fatalf("start tcp server failed: %v", err)
	}

	logrus.Infof("%v listen on: %v", proxy.cfg.Mode, proxy.cfg.ProxyAddress)

	for client := range clientChan {
		go func(client net.Conn) {
			remoteConn, err := utils.EasyTCPConnectServer(proxy.cfg.Secure.TLSEnableFlag.ConnectServerUseTLS,
				proxy.cfg.Secure.ConnectServerTLSConfig, proxy.cfg.RemoteProxyAddress, proxy.cfg.DialTimeout)
			if err != nil {
				logrus.Errorf("dial %v failed: %v", proxy.cfg.RemoteProxyAddress, err)
				return
			}

			defer client.Close()

			go func() {
				_, _ = io.Copy(client, remoteConn)
			}()
			_, _ = io.Copy(remoteConn, client)
		}(client)
	}
}
