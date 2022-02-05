// Copyright (c) 2022 Wireleap

package mrwclabels

type ContractLabels struct {
	Contract string `label:"contract"`
}

func (ct ContractLabels) SetContract(contract string) ContractLabels {
	ct.Contract = contract
	return ct
}

func (ct ContractLabels) GetConnection() ConnectionLabels {
	return ConnectionLabels{Contract: ct.Contract}
}

type ConnectionLabels struct {
	Contract string `label:"contract"`
	Origin   string `label:"origin"` // "client" or "target"
}

func (cl ConnectionLabels) SetContract(contract string) ConnectionLabels {
	cl.Contract = contract
	return cl
}

func (cl ConnectionLabels) GetContract() ContractLabels {
	return ContractLabels{Contract: cl.Contract}
}

func (cl ConnectionLabels) SetOrigin(origin string) ConnectionLabels {
	cl.Origin = origin
	return cl
}
