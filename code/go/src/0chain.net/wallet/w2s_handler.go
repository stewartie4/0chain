package wallet

var ConfirmTransaction FormSendHandler
var GetBlock FormSendHandler
var GetBalance FormSendHandler

func SetupW2SSenders() {
	ConfirmTransaction = SendFormHandler("v1/transaction/get/confirmation")
	GetBlock = SendFormHandler("v1/block/get")
	GetBalance = SendFormHandler("v1/client/get/balance")
}

func (c *Cluster) SendToSharders(handler SendHandler) Nodes {
	return c.Sharders.SendAtleast(len(c.Sharders), handler)
}
