package blademaster

import (
	"fmt"
	"strconv"
	"time"

	"kratos/pkg/ecode"
	"kratos/pkg/log"
	"kratos/pkg/net/metadata"

	"github.com/pkg/errors"
)

// Logger is logger  middleware
func Logger() HandlerFunc {
	const noUser = "no_user"
	return func(c *Context) {
		if c.Request.URL.String() == monitorPing {
			c.Next()
			return
		}

		now := time.Now()
		req := c.Request
		path := req.URL.Path
		params := req.Form
		var quota float64
		if deadline, ok := c.Context.Deadline(); ok {
			quota = time.Until(deadline).Seconds()
		}

		c.Next()

		err := c.Error
		cerr := ecode.Cause(err)

		// cause root error
		rootErr := errors.Cause(err)
		rootErrMsg := ""
		rootStack := ""
		if rootErr != nil {
			rootErrMsg = rootErr.Error()
			rootStack = fmt.Sprintf("%+v", rootErr)
		}

		dt := time.Since(now)
		caller := metadata.String(c, metadata.Caller)
		if caller == "" {
			caller = noUser
		}

		if len(c.RoutePath) > 0 {
			_metricServerReqCodeTotal.Inc(c.RoutePath[1:], caller, req.Method, strconv.FormatInt(int64(cerr.Code()), 10))
			_metricServerReqDur.Observe(int64(dt/time.Millisecond), c.RoutePath[1:], caller, req.Method)
		}

		lf := log.Infov
		errmsg := ""
		isSlow := dt >= (time.Millisecond * 500)
		if err != nil {
			errmsg = err.Error()
			if ecode.Equal(cerr, ecode.RequestErr) {
				errmsg = c.ErrorMsg
			}
			lf = log.Errorv
			if cerr.Code() > 0 {
				lf = log.Warnv
			}
		} else {
			if isSlow {
				lf = log.Warnv
			}
		}
		lf(c,
			log.KVString("method", req.Method),
			log.KVString("ip", c.RemoteIP()),
			log.KVString("user", caller),
			log.KVString("path", path),
			log.KVString("params", params.Encode()),
			log.KVInt("ret", cerr.Code()),
			log.KVString("msg", cerr.Message()),
			log.KVString("stack", fmt.Sprintf("%+v", err)),
			log.KVString("err", errmsg),
			log.KVFloat64("timeout_quota", quota),
			log.KVFloat64("ts", dt.Seconds()),
			log.KVString("source", "http-access-log"),
			log.KVString("authorization", req.Header.Get("Authorization")),
			log.KVString("sign", req.Header.Get("X-Sign")),
			log.KVString("root_err", rootErrMsg),
			log.KVString("root_err_stack", rootStack),
		)
	}
}
