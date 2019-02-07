package wallet

var RegisterClient EntitySendHandler
var SubmitTransaction EntitySendHandler
var CheckClientStatus FormSendHandler

func SetupW2MSenders() {
	RegisterClient = SendEntityHandler("v1/client/put")
	SubmitTransaction = SendEntityHandler("v1/transaction/put")
	CheckClientStatus = SendFormHandler("v1/client/get")
}

func (c *Cluster) SendToMiners(handler SendHandler) Nodes {
	return c.Miners.SendAtleast(len(c.Miners), handler)
}

func (c *Cluster) SendToSomeMiners(handler SendHandler, x int) Nodes {
	return c.Miners.SendAtleast(x, handler)
}

func (c *Cluster) SendTo(handler SendHandler) (bool, error) {
	return false, nil
}
