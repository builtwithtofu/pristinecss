package parser

type FontPaletteValuesAtRule struct {
	Name         []byte
	Declarations []Declaration
}

func (r *FontPaletteValuesAtRule) Type() NodeType { return NodeAtRule }
func (r *FontPaletteValuesAtRule) AtType() AtType { return AtFontPaletteValues }
func (r *FontPaletteValuesAtRule) String() string {
	return descriptorRuleString("FontPaletteValuesAtRule", r.Name, r.Declarations)
}
func (r *FontPaletteValuesAtRule) setName(v []byte)         { r.Name = v }
func (r *FontPaletteValuesAtRule) setDecls(v []Declaration) { r.Declarations = v }
func visitFontPaletteValuesAtRule(pv *ParseVisitor, node AtRule) {
	parseDashedDescriptorRule(pv, node.(*FontPaletteValuesAtRule))
}
