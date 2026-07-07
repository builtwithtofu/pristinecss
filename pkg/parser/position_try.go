package parser

type PositionTryAtRule struct {
	Name         []byte
	Declarations []Declaration
}

func (r *PositionTryAtRule) Type() NodeType { return NodeAtRule }
func (r *PositionTryAtRule) AtType() AtType { return AtPositionTry }
func (r *PositionTryAtRule) String() string {
	return descriptorRuleString("PositionTryAtRule", r.Name, r.Declarations)
}
func (r *PositionTryAtRule) setName(v []byte)         { r.Name = v }
func (r *PositionTryAtRule) setDecls(v []Declaration) { r.Declarations = v }
func visitPositionTryAtRule(pv *ParseVisitor, node AtRule) {
	parseDashedDescriptorRule(pv, node.(*PositionTryAtRule))
}
