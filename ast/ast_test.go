package ast

import (
	"github.com/Financial-Times/falco/token"
)

var T = token.Token{}

func comments(c ...string) Comments {
	cs := Comments{}
	for i := range c {
		cs = append(cs, &Comment{
			Value: c[i],
		})
	}
	return cs
}
