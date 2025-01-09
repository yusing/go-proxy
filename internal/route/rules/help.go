package rules

import "strings"

type Help struct {
	command     string
	description string
	args        map[string]string // args[arg] -> description
}

/*
Generate help string, e.g.

	 rewrite <from> <to>
		from: the path to rewrite, must start with /
		to: the path to rewrite to, must start with /
*/
func (h *Help) String() string {
	var sb strings.Builder
	sb.WriteString(h.command)
	sb.WriteString(" ")
	for arg := range h.args {
		sb.WriteRune('<')
		sb.WriteString(arg)
		sb.WriteString("> ")
	}
	if h.description != "" {
		sb.WriteString("\n\t")
		sb.WriteString(h.description)
		sb.WriteRune('\n')
	}
	sb.WriteRune('\n')
	for arg, desc := range h.args {
		sb.WriteRune('\t')
		sb.WriteString(arg)
		sb.WriteString(": ")
		sb.WriteString(desc)
		sb.WriteRune('\n')
	}
	return sb.String()
}
