package typecheck

import "voxlang/internal/source"

func (c *checker) pushScope() { c.scope = append(c.scope, map[string]varInfo{}) }
func (c *checker) popScope()  { c.scope = c.scope[:len(c.scope)-1] }

func (c *checker) scopeTop() map[string]varInfo {
	return c.scope[len(c.scope)-1]
}

func (c *checker) lookupVar(name string) (varInfo, bool) {
	for i := len(c.scope) - 1; i >= 0; i-- {
		if v, ok := c.scope[i][name]; ok {
			return v, true
		}
	}
	return varInfo{}, false
}

func (c *checker) errorAt(s source.Span, msg string) {
	fn, line, col := s.LocStart()
	c.diags.Add(fn, line, col, msg)
}
