package context

// Query 返回网址中查询键的值。
func (ctx *Context) Query(key string) string {
	v, _ := ctx.GetQuery(key)
	return v
}

// GetQuery 返回网址中查询键的值及存在状态。
func (ctx *Context) GetQuery(key string) (string, bool) {
	if vs, ok := ctx.Request.URL.Query()[key]; ok && len(vs) > 0 {
		return vs[0], true
	}

	return "", false
}
