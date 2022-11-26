// Copyright (c) 2022 Wireleap

package labels

type Contract struct {
	Contract string `label:"contract"`
	Role     string `label:"role"`
}

func (ct Contract) SetContract(contract string) Contract {
	ct.Contract = contract
	return ct
}

func (ct Contract) SetRole(role string) Contract {
	ct.Role = role
	return ct
}

func (ct Contract) GetConnection() Connection {
	return Connection{
		Contract: ct.Contract,
		Role:     ct.Role,
	}
}

func (ct Contract) WithErr(error string) ContractErr {
	return ContractErr{
		Contract: ct.Contract,
		Role:     ct.Role,
		Error:    error,
	}
}

func (ct Contract) WithCapMode(mode string) ContractNetCap {
	return ContractNetCap{
		Contract: ct.Contract,
		Role:     ct.Role,
		Mode:     mode,
	}
}

type ContractErr struct {
	Contract string `label:"contract"`
	Role     string `label:"role"`
	Error    string `label:"error"`
}

func (cl ContractErr) SetContract(contract string) ContractErr {
	cl.Contract = contract
	return cl
}

func (cl ContractErr) SetRole(role string) ContractErr {
	cl.Role = role
	return cl
}

func (cl ContractErr) GetContract() Contract {
	return Contract{
		Contract: cl.Contract,
		Role:     cl.Role,
	}
}

func (cl ContractErr) SetError(error string) ContractErr {
	cl.Error = error
	return cl
}

type ContractNetCap struct {
	Contract string `label:"contract"`
	Role     string `label:"role"`
	Mode     string `label:"mode"`
}

func (cnc ContractNetCap) SetContract(contract string) ContractNetCap {
	cnc.Contract = contract
	return cnc
}

func (cnc ContractNetCap) SetRole(role string) ContractNetCap {
	cnc.Role = role
	return cnc
}

func (cnc ContractNetCap) GetContract() Contract {
	return Contract{
		Contract: cnc.Contract,
		Role:     cnc.Role,
	}
}

func (cnc ContractNetCap) SetCapMode(mode string) ContractNetCap {
	cnc.Mode = mode
	return cnc
}

type Connection struct {
	Contract string `label:"contract"`
	Role     string `label:"role"`
	Origin   string `label:"origin"` // "client" or "target"
}

func (cl Connection) SetContract(contract string) Connection {
	cl.Contract = contract
	return cl
}

func (cl Connection) SetRole(role string) Connection {
	cl.Role = role
	return cl
}

func (cl Connection) GetContract() Contract {
	return Contract{
		Contract: cl.Contract,
		Role:     cl.Role,
	}
}

func (cl Connection) SetOrigin(origin string) Connection {
	cl.Origin = origin
	return cl
}
