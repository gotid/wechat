package context

import "encoding/xml"

// String 提供字符串响应流
func (ctx *Context) String(s string) {
	ctx.SetContentTypeText()
	ctx.Render([]byte(s))
}

// XML 提供 XML 响应流
func (ctx *Context) XML(v interface{}) {
	ctx.SetContentTypeXML()
	bs, err := xml.Marshal(v)
	if err != nil {
		panic(err)
	}
	ctx.Render(bs)
}

// Render 提供 http 响应流
func (ctx *Context) Render(bs []byte) {
	ctx.Writer.WriteHeader(200)
	_, err := ctx.Writer.Write(bs)
	if err != nil {
		panic(err)
	}
}

// SetContentTypeText 设置 http 响应内容类型为纯文本
func (ctx *Context) SetContentTypeText() {
	ctx.SetContentType([]string{"text/plain; charset=utf-8"})
}

// SetContentTypeXML 设置 http 响应内容类型为 XML
func (ctx *Context) SetContentTypeXML() {
	ctx.SetContentType([]string{"application/xml; charset=utf-8"})
}

// SetContentType 设置 http 响应内容类型
func (ctx *Context) SetContentType(vs []string) {
	h := ctx.Writer.Header()
	if vs := h["Content-Type"]; len(vs) == 0 {
		h["Content-Type"] = vs
	}
}
