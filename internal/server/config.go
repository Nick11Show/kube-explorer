package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/apiserver/pkg/urlbuilder"
	steveauth "github.com/rancher/steve/pkg/auth"
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/rancher/steve/pkg/server/cli"
	"github.com/rancher/steve/pkg/server/router"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/cnrancher/kube-explorer/internal/config"
	"github.com/cnrancher/kube-explorer/internal/resources/cluster"
	"github.com/cnrancher/kube-explorer/internal/ui"
	"github.com/cnrancher/kube-explorer/internal/version"
)

func ToServer(ctx context.Context, c *cli.Config, sqlCache bool) (*server.Server, error) {
	var (
		auth steveauth.Middleware
	)

	restConfig, err := kubeconfig.GetNonInteractiveClientConfigWithContext(c.KubeConfig, c.Context).ClientConfig()
	if err != nil {
		return nil, err
	}
	restConfig.RateLimiter = ratelimit.None

	restConfig.Insecure = config.InsecureSkipTLSVerify
	if restConfig.Insecure {
		restConfig.CAData = nil
		restConfig.CAFile = ""
	}

	if config.Token != "" {
		// Use the provided bearer token as the sole credential for talking to
		// the kube-apiserver. The server URL and CA still come from the
		// kubeconfig (--kubeconfig / auto-detected RKE2-K3s path).
		restConfig.BearerToken = config.Token
		restConfig.BearerTokenFile = ""
		restConfig.Username = ""
		restConfig.Password = ""
		restConfig.CertData = nil
		restConfig.KeyData = nil
		restConfig.CertFile = ""
		restConfig.KeyFile = ""
	}

	if c.WebhookConfig.WebhookAuthentication {
		auth, err = c.WebhookConfig.WebhookMiddleware()
		if err != nil {
			return nil, err
		}
	}

	// steve v0.9 exposes --enable-header-auth, but kube-explorer builds its
	// server through this custom ToServer rather than cli.Config.ToServer, so
	// the flag would otherwise be accepted and silently ignored. Mirror the
	// upstream behaviour: honour Impersonate-* headers, and fall back to an
	// admin identity when they are absent.
	if c.EnableHeaderAuthentication {
		impersonateOrAdmin := func(req *http.Request) (user.Info, bool, error) {
			info, ok, err := steveauth.Impersonation(req)
			if ok || err != nil {
				return info, ok, err
			}
			return steveauth.AlwaysAdmin(req)
		}
		auth = steveauth.ToMiddleware(steveauth.AuthenticatorFunc(impersonateOrAdmin))
	}

	controllers, err := server.NewController(restConfig, nil)
	if err != nil {
		return nil, err
	}

	ui, apiui := ui.New(&ui.Options{
		ReleaseSetting: version.IsRelease,
		Path:           func() string { return c.UIPath },
	})

	steveServer, err := server.New(ctx, restConfig, &server.Options{
		AuthMiddleware: auth,
		Controllers:    controllers,
		Next:           ui,
		SQLCache:       sqlCache,
		// router needs to hack here
		Router: func(h router.Handlers) http.Handler {
			return handleProxyHeader(
				rewriteLocalCluster(
					router.Routes(h),
				),
			)
		},
	})
	if err != nil {
		return nil, err
	}

	steveServer.APIServer.CustomAPIUIResponseWriter(apiui.CSS(), apiui.JS(), func() string { return config.APIUIVersion })

	// registrer local cluster
	if err := cluster.Register(ctx, steveServer, c.Context); err != nil {
		return steveServer, err
	}
	// wrap default store
	steveServer.SchemaFactory.AddTemplate(schema.Template{
		Customize: func(a *types.APISchema) {
			if a.Store == nil {
				return
			}
			a.Store = &deleteOptionStore{
				Store: a.Store,
			}
		},
	})
	return steveServer, controllers.Start(ctx)
}

func rewriteLocalCluster(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/k8s/clusters/local") {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/k8s/clusters/local")
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		}
		next.ServeHTTP(rw, req)
	})
}

func handleProxyHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if value := req.Header.Get("X-Forwarded-Prefix"); value != "" {
			req.Header.Set(urlbuilder.PrefixHeader, value)
		}
		next.ServeHTTP(rw, req)
	})
}
