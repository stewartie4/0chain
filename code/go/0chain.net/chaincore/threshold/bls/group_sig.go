package bls

import (
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"fmt"
	"github.com/herumi/bls/ffi/go/bls"
	"go.uber.org/zap"
)

/*VerificationKey - Is of type bls.PublicKey*/
type VerificationKey = bls.PublicKey

/*SimpleBLS - to manage BLS process */
type SimpleBLS struct {
	T                int
	N                int
	Msg              Message
	SigShare         Sign
	gpPubKey         GroupPublicKey
	SecKeyShareGroup Key
	GpSign           Sign
	ID               PartyID
}

/*MakeSimpleBLS - to create bls object */
func MakeSimpleBLS(dkg *DKG) SimpleBLS {
	bs := SimpleBLS{
		T:        dkg.T,
		N:        dkg.N,
		Msg:      " ",
		SigShare: Sign{},
		gpPubKey: dkg.GpPubKey,

		SecKeyShareGroup: dkg.SecKeyShareGroup,
		GpSign:           Sign{},
		ID:               dkg.ID,
	}
	return bs

}

/*SignMsg - Bls sign share is computed by signing the message r||RBO(r-1) with secret key share of group of that party */
func (bs *SimpleBLS) SignMsg() Sign {

	aggSecKey := bs.SecKeyShareGroup
	sigShare := *aggSecKey.Sign(bs.Msg)
	return sigShare
}

/*RecoverGroupSig - To compute the Gp sign with any k number of BLS sig shares */
func (bs *SimpleBLS) RecoverGroupSig(from []PartyID, shares []Sign) Sign {

	signVec := shares
	idVec := from

	var sig Sign
	err := sig.Recover(signVec, idVec)

	if err == nil {
		bs.GpSign = sig
		return sig
	}

	VRFLogger.Error("Recover Gp Sig not done", zap.Error(err))

	return sig

}

// VerifyVrf verify received sigShare with the vvec
func VerifyVrf(sigShare string, senderId string, senderIndex int, msgString string, vvec []bls.PublicKey) error {
	VRFLogger.Info("VerifyVrf", zap.String("msgString", msgString), zap.Int("senderIndex", senderIndex), zap.String("senderId", senderId), zap.String("sigShare", sigShare))

	if vvec == nil || len(vvec) == 0 {
		return common.NewError("vrfverify_vvec_empty_err", fmt.Sprintf("No vvec yet. Could not verify the signedshare: %v. ", sigShare))
	}

	var signedShare Sign
	err := signedShare.SetHexString(sigShare)

	if err != nil {
		VRFLogger.Error("failed to convert sigShare to Sign", zap.String("sigShare", sigShare))
		return err
	}

	var forID bls.ID
	err = forID.SetDecString(senderId)
	if err != nil {
		VRFLogger.Error("failed to convert partyId from senderId", zap.String("senderId", senderId))
		return err
	}

	VRFLogger.Info("ComputeIDdkg", zap.Any("partyID", forID.GetDecString()), zap.Int("index", senderIndex))

	var pubK bls.PublicKey
	err = pubK.Set(vvec, &forID)
	if err != nil {
		VRFLogger.Info("VerifyVrf Sender is not ok", zap.Any("ID", forID))
		return err
	}

	VRFLogger.Info("VerifyVrf Sender is ok. Checking message", zap.String("sigShare", sigShare), zap.String("msgString", msgString))

	var msg Message
	msg = msgString
	if !signedShare.Verify(&pubK, msg) {
		VRFLogger.Info("VerifyVrf Message failed")
		return common.NewError("vrf_verification_err", fmt.Sprintf("Could not verify the signedshare: %v", sigShare))
	}
	VRFLogger.Info("VerifyVrf is success!")

	return nil
}

// CalcRandomBeacon - Calculates the random beacon output
func (bs *SimpleBLS) CalcRandomBeacon(recSig []string, recIDs []string) string {

	VRFLogger.Debug("Threshold number of bls sig shares are received ...")
	bs.CalBlsGpSign(recSig, recIDs)
	rboOutput := encryption.Hash(bs.GpSign.GetHexString())
	return rboOutput
}

// CalBlsGpSign - The function calls the RecoverGroupSig function which calculates the Gp Sign
func (bs *SimpleBLS) CalBlsGpSign(recSig []string, recIDs []string) {
	//Move this to group_sig.go
	signVec := make([]Sign, 0)
	var signShare Sign

	for i := 0; i < len(recSig); i++ {
		err := signShare.SetHexString(recSig[i])

		if err == nil {
			signVec = append(signVec, signShare)
		} else {
			VRFLogger.Error("signVec not computed correctly", zap.Error(err))
		}
	}

	idVec := make([]PartyID, 0)
	var forID PartyID
	for i := 0; i < len(recIDs); i++ {
		err := forID.SetDecString(recIDs[i])
		if err == nil {
			idVec = append(idVec, forID)
		}
	}
	/*
		VRFLogger.Debug("Printing bls shares and respective party IDs who sent used for computing the Gp Sign")

		for _, sig := range signVec {
			VRFLogger.Debug(" Printing bls shares", zap.Any("sig_shares", sig.GetHexString()))

		}
		for _, fromParty := range idVec {
			VRFLogger.Debug(" Printing party IDs", zap.Any("from_party", fromParty.GetHexString()))

		}
	*/
	bs.RecoverGroupSig(idVec, signVec)

}

/*CalcGroupsVvec - Aggregates the committed verification vectors by all partys to get the Groups Vvec */
func CalcGroupsVvec(Vvecs [][]VerificationKey, t int, n int) []VerificationKey {

	groupsVvec := make([]VerificationKey, t)

	for i := range Vvecs {

		for j := range Vvecs[i] {

			pub2 := Vvecs[i][j]
			pub1 := groupsVvec[j]
			pub1.Add(&pub2)
			groupsVvec[j] = pub1
		}
	}
	return groupsVvec
}
