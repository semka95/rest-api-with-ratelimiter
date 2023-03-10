package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"rate-limiter/limiter"
	"rate-limiter/templates"

	"go.uber.org/zap"
)

// API represents rest api
type API struct {
	ipMask      net.IPMask
	rateLimiter limiter.RequestLimiter
	logger      *zap.Logger
}

// NewRouter creates api router
func (a *API) NewRouter(ipMask net.IPMask, rateLimiter limiter.RequestLimiter, logger *zap.Logger) *http.ServeMux {
	a.ipMask = ipMask
	a.rateLimiter = rateLimiter
	a.logger = logger

	mux := http.NewServeMux()
	mux.Handle("/", a.rateLimiterMiddleware(http.HandlerFunc(a.homePage)))
	mux.Handle("/reset", http.HandlerFunc(a.resetLimit))

	return mux
}

// GET / - home page
func (a *API) homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the home page!")
}

// rateLimiterMiddleware is middleware that limits number of currently processed requests
// at a time per subnet.
func (a *API) rateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// parsing IP address from request
		ipStr := req.RemoteAddr
		fwdAddress := req.Header.Get("X-Forwarded-For")
		fwdAddress = strings.TrimSpace(fwdAddress)
		if fwdAddress != "" {
			ipStr = fwdAddress
			ips := strings.Split(fwdAddress, ",")
			if len(ips) > 1 {
				ipStr = ips[0]
			}
		}
		ipAddr := net.ParseIP(ipStr)
		if ipAddr == nil {
			http.Error(rw, fmt.Sprintf("wrong ip: %s", ipStr), http.StatusBadRequest)
			a.logger.Warn("bad ip address", zap.String("ip", ipStr))
			return
		}

		// masking IP to get subnet and checking if it already timed out
		maskedIP := ipAddr.Mask(a.ipMask).String()
		if a.rateLimiter.IsTimedOut(maskedIP) {
			t, reqLimit, interval, err := a.rateLimiter.Get(maskedIP)
			if err != nil {
				http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			rw.Header().Add("Retry-After", t.Format(http.TimeFormat))

			rw.WriteHeader(http.StatusTooManyRequests)
			templates.TooManyReqTmpl.Execute(rw, templates.TooManyReqData{Requests: reqLimit, Duration: interval, Date: t.Format(http.TimeFormat)})
			a.logger.Debug("too many requests", zap.String("ip", ipAddr.String()), zap.String("subnet", maskedIP))
			return
		}

		// taking token per subnet
		remaining, ok, err := a.rateLimiter.Take(req.Context(), maskedIP)
		if err != nil {
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			a.logger.Error("can't take limiter token", zap.Error(err), zap.String("ip", ipAddr.String()), zap.String("subnet", maskedIP))
			return
		}
		// TODO: maybe unnecessary check
		if !ok {
			http.Error(rw, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			a.logger.Debug("too many requests", zap.String("ip", ipAddr.String()), zap.String("subnet", maskedIP))
			return
		}

		// if there is no tokens left, reject all requests from subnet
		if remaining == 0 {
			a.logger.Info("subnet cooldown", zap.String("subnet", maskedIP))
			a.rateLimiter.CooldownSubnet(maskedIP)
		}
		next.ServeHTTP(rw, req)
	})
}

func (a *API) resetLimit(w http.ResponseWriter, r *http.Request) {
	subnet := r.URL.Query().Get("ip")
	a.logger.Debug("reset ip", zap.String("ip", subnet))
	if subnet == "" {
		http.Error(w, fmt.Sprintf("wrong ip: %s", subnet), http.StatusBadRequest)
		a.logger.Warn("bad ip address", zap.String("ip", subnet))
		return
	}

	err := a.rateLimiter.Reset(r.Context(), subnet)
	if err != nil {
		http.Error(w, fmt.Sprintf("wrong ip: %s", subnet), http.StatusBadRequest)
		a.logger.Warn("no such subnet", zap.String("ip", subnet))
		return
	}

	w.WriteHeader(http.StatusOK)
}
