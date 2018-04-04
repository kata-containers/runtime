package api

type AgentConfig struct {
	// ConnProto: the method to connect to agent(or proxy)
	// "unix": the paths are unix socket
	// "vsock": use cid and port pair to connect to the agent (kata-only)
	// "kvmtool": the paths are kvmtool driver created pty slaver with escaping character
	ConnProto string
	// AgentProto: the agent protocol
	// "hyperstart": orginal hyperstart protocol
	// "kata": kata agent protocol
	// "kata-yamux": kata agent protocol over yamux
	AgentProto string

	HyperstartCtlPath, HyperstartStreamPath string
	KataPath                                string
	KataVsockCid, KataVsockPort             uint32
}

type Agent interface {
	Config() AgentConfig

	// methods are private in the lib
}

type AgentBuilder interface {
	Build(AgentConfig) Agent
}

// return the default AgentBuilder provided by the lib
func DirectConnect() AgentBuilder {
}

// Example code for how to use AgentBuilder with proxy
// The way how to luanch proxy is user (an oci-runtime-cli implementation) specific.
// So this code can be put in runtime code.
//
// Proxy is an implementation of kata-agent-api, it fits for AgentBuilder
//
// struct proxyAgent {
//    proxyBinary string
//    // other fields
// }
// func (p proxyAgent) Build(configToVM AgentConfig) Agent {
//    var configToProxy AgentConfig
//    configToProxy, err := p.launchProxy(configToVM)
//    if err != nil { return nil }
//    return DirectConnect().Build(configToProxy)
// }
