package templates

import (
	"html/template"
	"time"
)

// TooManyReqTmpl is html template for too many requests http response
var TooManyReqTmpl = template.Must(template.New("tooManyReqs").Parse(tpl429Response))

// TooManyReqData represents data for TooManyReqTmpl template
type TooManyReqData struct {
	Requests uint64
	Duration time.Duration
	Date     string
}

const tpl429Response = `
<html>
	<head>
		<title>Too Many Requests</title>
	</head>
	<body>
		<h1>Too Many Requests</h1>
		<p>I only allow {{.Requests}} requests per {{.Duration}} to this Web site per
		subnet. Try again after {{.Date}}.</p>
	</body>
</html>
`
