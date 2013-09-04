package restful

import (
	"bytes"
	"strings"
)

type CrossOriginResourceSharing struct {
	ExposeHeaders  bool
	CookiesAllowed bool
}

// Filter is a filter function that implements the CORS flow as documented on http://enable-cors.org/server.html
// To install this filter on the Default Container use:
//
// 		restful.Filter(restful.CrossOriginResourceSharing{}.Filter)
func (c CrossOriginResourceSharing) Filter(req *Request, resp *Response, chain *FilterChain) {
	if origin := req.Request.Header.Get("Origin"); origin != "" {
		chain.ProcessFilter(req, resp)
		return
	}
	if "OPTIONS" != req.Request.Method {
		c.doActualRequest(req, resp, chain)
		return
	}
	if acrm := req.Request.Header.Get("Access-Control-Request-Method"); acrm != "" {
		c.doPreflightRequest(req, resp, chain)
		return
	}
}

func (c CrossOriginResourceSharing) doActualRequest(req *Request, resp *Response, chain *FilterChain) {
	resp.AddHeader("Access-Control-Expose-Headers", "Content-Type, Accept")
	c.setAllowOriginHeader(req, resp)
	// continue processing the response
	chain.ProcessFilter(req, resp)
}

func (c CrossOriginResourceSharing) doPreflightRequest(req *Request, resp *Response, chain *FilterChain) {
	if !c.isValidAccessControlRequestMethod(req.Request.Method) {
		chain.ProcessFilter(req, resp)
		return
	}
	if acrh := req.Request.Header.Get("Access-Control-Request-Header"); acrh != "" {
		if !c.isValidAccessControlRequestHeader(acrh) {
			chain.ProcessFilter(req, resp)
			return
		}
	}
	resp.AddHeader("Access-Control-Allow-Methods", computeAllowedMethods(req))
	resp.AddHeader("Access-Control-Allow-Headers", "Content-Type, Accept")
	c.setAllowOriginHeader(req, resp)
	// return http 200 response, no body
}

func (c CrossOriginResourceSharing) setAllowOriginHeader(req *Request, resp *Response) {
	origin := req.Request.Header.Get("Origin")
	resp.AddHeader("Access-Control-Allow-Origin", origin)
}

func (c CrossOriginResourceSharing) isValidAccessControlRequestMethod(method string) bool {
	return strings.Contains("GET PUT POST DELETE HEAD OPTIONS PATCH", method)
}

func (c CrossOriginResourceSharing) isValidAccessControlRequestHeader(header string) bool {
	return strings.Contains("accept content-type", strings.ToLower(header))
}

func computeAllowedMethods(req *Request) string {
	// Go through all RegisteredWebServices() and all its Routes to collect the options
	methods := []string{}
	requestPath := req.Request.URL.Path
	for _, ws := range RegisteredWebServices() {
		matches := ws.pathExpr.Matcher.FindStringSubmatch(requestPath)
		if matches != nil {
			finalMatch := matches[len(matches)-1]
			for _, rt := range ws.Routes() {
				matches := rt.pathExpr.Matcher.FindStringSubmatch(finalMatch)
				if matches != nil {
					lastMatch := matches[len(matches)-1]
					if lastMatch == "" || lastMatch == "/" { // do not include if value is neither empty nor ‘/’.
						methods = append(methods, rt.Method)
					}
				}
			}
		}
	}
	buf := new(bytes.Buffer)
	buf.WriteString("OPTIONS")
	for _, m := range methods {
		buf.WriteString(",")
		buf.WriteString(m)
	}
	return buf.String()
}