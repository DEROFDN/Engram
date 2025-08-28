package main

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"github.com/civilware/Gnomon/structures"
	"github.com/civilware/epoch"
	"github.com/civilware/tela"
	"github.com/creachadair/jrpc2/handler"
)

// Further methods to add to XSWD,
// Gnomon. methods passthrough and do not require permission
var EngramHandler = map[string]handler.Func{
	"GetPrimaryUsername":                         handler.New(GetPrimaryUsername),
	"Gnomon.GetLastIndexHeight":                  handler.New(GetLastIndexHeight),
	"Gnomon.GetTxCount":                          handler.New(GetTxCount),
	"Gnomon.GetOwner":                            handler.New(GetOwner),
	"Gnomon.GetAllOwnersAndSCIDs":                handler.New(GetAllOwnersAndSCIDs),
	"Gnomon.GetAllNormalTxWithSCIDByAddr":        handler.New(GetAllNormalTxWithSCIDByAddr),
	"Gnomon.GetAllNormalTxWithSCIDBySCID":        handler.New(GetAllNormalTxWithSCIDBySCID),
	"Gnomon.GetAllSCIDInvokeDetails":             handler.New(GetAllSCIDInvokeDetails),
	"Gnomon.GetAllSCIDInvokeDetailsByEntrypoint": handler.New(GetAllSCIDInvokeDetailsByEntrypoint),
	"Gnomon.GetAllSCIDInvokeDetailsBySigner":     handler.New(GetAllSCIDInvokeDetailsBySigner),
	"Gnomon.GetGetInfoDetails":                   handler.New(GetGetInfoDetails),
	"Gnomon.GetSCIDVariableDetailsAtTopoheight":  handler.New(GetSCIDVariableDetailsAtTopoheight),
	"Gnomon.GetAllSCIDVariableDetails":           handler.New(GetAllSCIDVariableDetails),
	"Gnomon.GetSCIDKeysByValue":                  handler.New(GetSCIDKeysByValue),
	"Gnomon.GetSCIDValuesByKey":                  handler.New(GetSCIDValuesByKey),
	"Gnomon.GetLiveSCIDKeysByValue":              handler.New(GetLiveSCIDKeysByValue),
	"Gnomon.GetLiveSCIDValuesByKey":              handler.New(GetLiveSCIDValuesByKey),
	"Gnomon.GetSCIDInteractionHeight":            handler.New(GetSCIDInteractionHeight),
	"Gnomon.GetInteractionIndex":                 handler.New(GetInteractionIndex),
	"Gnomon.GetInvalidSCIDDeploys":               handler.New(GetInvalidSCIDDeploys),
	"Gnomon.GetAllMiniblockDetails":              handler.New(GetAllMiniblockDetails),
	"Gnomon.GetMiniblockDetailsByHash":           handler.New(GetMiniblockDetailsByHash),
	"Gnomon.GetMiniblockCountByAddress":          handler.New(GetMiniblockCountByAddress),
	"Gnomon.GetSCIDInteractionByAddr":            handler.New(GetSCIDInteractionByAddr),
}

type SCID_Param struct {
	SCID string `json:"scid"`
}

type Address_Param struct {
	Address string `json:"address"`
}

// GetPrimaryUsername result
type Username_Result struct {
	Username string `json:"username"`
}

// GetPrimaryUsername gets primary username from wallet
func GetPrimaryUsername(ctx context.Context) (result Username_Result, err error) {
	if session.Username != "" {
		var address string
		address, err = engram.Disk.NameToAddress(session.Username)
		if err != nil {
			return
		}

		if engram.Disk.GetAddress().String() != address {
			err = fmt.Errorf("could not validate primary username")
			return
		}

		result.Username = session.Username
	} else {
		err = fmt.Errorf("could not get primary username")
	}

	return
}

// HandleTELALinks
type (
	TELALink_Params struct {
		TelaLink string `json:"telaLink"` // format is target://<arg>/<arg>/...
	}

	// TELALink_Display is used internally when Engram processes the TELALink_Params
	TELALink_Display struct {
		Name     string              `json:"nameHdr,omitempty"`
		Descr    string              `json:"descrHdr,omitempty"`
		DURL     string              `json:"dURL,omitempty"`
		TelaLink string              `json:"telaLink"` // format is target://<arg>/<arg>/...
		Rating   *tela.Rating_Result `json:"rating,omitempty"`
	}

	TELALink_Result struct {
		TelaLinkResult string `json:"telaLinkResult"`
	}
)

// HandleTELALinks parses and handles all TELA links
func HandleTELALinks(ctx context.Context, p TELALink_Params) (result TELALink_Result, err error) {
	if gnomon.Index == nil {
		// Match Engram behavior as it disables TELA when Gnomon is inactive
		err = fmt.Errorf("gnomon is not active")
		return
	}

	target, args, err := tela.ParseTELALink(p.TelaLink)
	if err != nil {
		err = fmt.Errorf("could not parse tela link: %s", err)
		return
	}

	switch target {
	case "tela":
		switch args[0] {
		case "open":
			var link string
			link, err = tela.OpenTELALink(p.TelaLink, session.Daemon)
			if err != nil {
				return
			}

			var url *url.URL
			url, err = url.Parse(link)
			if err != nil {
				err = fmt.Errorf("could not parse URL: %s", err)
				return
			}

			err = fyne.CurrentApp().OpenURL(url)
			if err != nil {
				err = fmt.Errorf("could not open tela link: %s", err)
				return
			}

			result.TelaLinkResult = link
		default:
			err = fmt.Errorf("unknown tela argument %q", args[0])
		}
	case "engram":
		switch args[0] {
		case "asset":
			if len(args) < 2 {
				err = fmt.Errorf("%q has invalid tela link format for engram://asset", p.TelaLink)
				return
			}

			switch args[1] {
			case "manager":
				if len(args) < 3 || len(args[2]) != 64 {
					err = fmt.Errorf("%q has invalid scid argument for engram://asset/manager", p.TelaLink)
					return
				}

				session.LastDomain = session.Window.Content()
				session.Window.SetContent(layoutTransition())
				session.Window.SetContent(layoutAssetManager(args[2]))
				removeOverlays()

				result.TelaLinkResult = fmt.Sprintf("%s %s %s", target, args[0], args[1])
			default:
				err = fmt.Errorf("unknown engram://asset argument %q", args[1])
			}
		default:
			err = fmt.Errorf("unknown engram argument %q", args[0])
		}
	default:
		err = fmt.Errorf("unknown target %q", target)
	}

	return
}

// EPOCH attempt params with address for the caller to define
type Attempt_With_Address_Params struct {
	Hashes  int    `json:"hashes"`
	Address string `json:"address"`
}

// AttemptEPOCHWithAddr is intended to be used for dApps to set a single use address for AttemptEPOCH calls,
// it will start/stop a new GetWork connection each time it is called, address param can be a DERO name and will be converted
func AttemptEPOCHWithAddr(ctx context.Context, p Attempt_With_Address_Params) (result epoch.EPOCH_Result, err error) {
	if p.Address == "" {
		err = fmt.Errorf("address param cannot be empty")
		return
	}

	if len(p.Address) < 66 {
		var address string
		address, err = engram.Disk.NameToAddress(p.Address)
		if err != nil {
			err = fmt.Errorf("could not match name to DERO address: %s", err)
			return
		}

		p.Address = address
	}

	maxHashes := epoch.GetMaxHashes()
	if p.Hashes > maxHashes {
		err = fmt.Errorf("hashes exceeds maxHashes %d/%d", p.Hashes, maxHashes)
		return
	}

	err = epoch.StartGetWork(p.Address, session.Daemon)
	if err != nil {
		return
	}

	defer epoch.StopGetWork()

	err = epoch.JobIsReady(time.Second * 10)
	if err != nil {
		return
	}

	result, err = epoch.AttemptHashes(p.Hashes)
	if result.Hashes > 0 {
		storeEPOCHTotal(time.Second * 2)
	}

	return
}

// GetLastIndexHeight
type GetLastIndexHeight_Result struct {
	LastIndexHeight int64 `json:"lastIndexHeight"`
}

// GetLastIndexHeight from Gnomon's gravdb
func GetLastIndexHeight(ctx context.Context) (result GetLastIndexHeight_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var height int64
	switch gnomon.Index.DBType {
	case "gravdb":
		height, err = gnomon.Index.GravDBBackend.GetLastIndexHeight()
	case "boltdb":
		height, err = gnomon.Index.BBSBackend.GetLastIndexHeight()
	}
	if err != nil {
		return
	}

	result.LastIndexHeight = height

	return
}

// GetTxCount
type (
	GetTxCount_Params struct {
		TxType string `json:"txType"`
	}

	GetTxCount_Result struct {
		TxCount int64 `json:"txCount"`
	}
)

// GetTxCount from Gnomon's gravdb
func GetTxCount(ctx context.Context, p GetTxCount_Params) (result GetTxCount_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var count int64

	switch gnomon.Index.DBType {
	case "gravdb":
		count = gnomon.Index.GravDBBackend.GetTxCount(p.TxType)
	case "boltdb":
		count = gnomon.Index.BBSBackend.GetTxCount(p.TxType)
	}

	result.TxCount = count

	return
}

// GetOwner
type GetOwner_Result struct {
	Owner string `json:"getOwner"`
}

// GetOwner of scid from Gnomon's gravdb
func GetOwner(ctx context.Context, p SCID_Param) (result GetOwner_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var owner string

	switch gnomon.Index.DBType {
	case "gravdb":
		owner = gnomon.Index.GravDBBackend.GetOwner(p.SCID)
	case "boltdb":
		owner = gnomon.Index.BBSBackend.GetOwner(p.SCID)
	}
	if owner == "" {
		err = fmt.Errorf("no stored owner for %s", p.SCID)
		return
	}

	result.Owner = owner

	return
}

// GetAllOwnersAndSCIDs
type GetAllOwnersAndSCIDs_Result struct {
	AllOwners map[string]string `json:"allOwners"`
}

// GetAllOwnersAndSCIDs from Gnomon's gravdb
func GetAllOwnersAndSCIDs(ctx context.Context) (result GetAllOwnersAndSCIDs_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	owners := make(map[string]string)

	switch gnomon.Index.DBType {
	case "gravdb":
		owners = gnomon.Index.GravDBBackend.GetAllOwnersAndSCIDs()
	case "boltdb":
		owners = gnomon.Index.BBSBackend.GetAllOwnersAndSCIDs()
	}

	result.AllOwners = owners

	return
}

// GetAllNormalTxWithSCID
type GetAllNormalTxWithSCID_Result struct {
	NormalTxWithSCID []*structures.NormalTXWithSCIDParse `json:"normalTxWithSCID"`
}

// GetAllNormalTxWithSCIDByAddr from Gnomon's gravdb
func GetAllNormalTxWithSCIDByAddr(ctx context.Context, p Address_Param) (result GetAllNormalTxWithSCID_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var owners []*structures.NormalTXWithSCIDParse

	switch gnomon.Index.DBType {
	case "gravdb":
		owners = gnomon.Index.GravDBBackend.GetAllNormalTxWithSCIDByAddr(p.Address)
	case "boltdb":
		owners = gnomon.Index.BBSBackend.GetAllNormalTxWithSCIDByAddr(p.Address)
	}

	result.NormalTxWithSCID = owners

	return
}

// GetAllNormalTxWithSCIDBySCID from Gnomon's gravdb
func GetAllNormalTxWithSCIDBySCID(ctx context.Context, p SCID_Param) (result GetAllNormalTxWithSCID_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var owners []*structures.NormalTXWithSCIDParse

	switch gnomon.Index.DBType {
	case "gravdb":
		owners = gnomon.Index.GravDBBackend.GetAllNormalTxWithSCIDBySCID(p.SCID)
	case "boltdb":
		owners = gnomon.Index.BBSBackend.GetAllNormalTxWithSCIDBySCID(p.SCID)
	}

	result.NormalTxWithSCID = owners

	return
}

// GetAllSCIDInvokeDetails
type (
	GetAllSCIDInvokeDetails_Params struct {
		SCID       string `json:"scid"`
		Entrypoint string `json:"entrypoint"`
		Signer     string `json:"signer"`
	}

	GetAllSCIDInvokeDetails_Result struct {
		Invokes []*structures.SCTXParse `json:"invokeDetails"`
	}
)

// GetAllSCIDInvokeDetails from Gnomon's gravdb
func GetAllSCIDInvokeDetails(ctx context.Context, p SCID_Param) (result GetAllSCIDInvokeDetails_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var invokes []*structures.SCTXParse

	switch gnomon.Index.DBType {
	case "gravdb":
		invokes = gnomon.Index.GravDBBackend.GetAllSCIDInvokeDetails(p.SCID)
	case "boltdb":
		invokes = gnomon.Index.BBSBackend.GetAllSCIDInvokeDetails(p.SCID)
	}

	result.Invokes = invokes

	return
}

// GetAllSCIDInvokeDetailsByEntrypoint from Gnomon's gravdb
func GetAllSCIDInvokeDetailsByEntrypoint(ctx context.Context, p GetAllSCIDInvokeDetails_Params) (result GetAllSCIDInvokeDetails_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var invokes []*structures.SCTXParse

	switch gnomon.Index.DBType {
	case "gravdb":
		invokes = gnomon.Index.GravDBBackend.GetAllSCIDInvokeDetailsByEntrypoint(p.SCID, p.Entrypoint)
	case "boltdb":
		invokes = gnomon.Index.BBSBackend.GetAllSCIDInvokeDetailsByEntrypoint(p.SCID, p.Entrypoint)
	}

	result.Invokes = invokes

	return
}

// GetAllSCIDInvokeDetailsBySigner from Gnomon's gravdb
func GetAllSCIDInvokeDetailsBySigner(ctx context.Context, p GetAllSCIDInvokeDetails_Params) (result GetAllSCIDInvokeDetails_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var invokes []*structures.SCTXParse

	switch gnomon.Index.DBType {
	case "gravdb":
		invokes = gnomon.Index.GravDBBackend.GetAllSCIDInvokeDetailsBySigner(p.SCID, p.Signer)
	case "boltdb":
		invokes = gnomon.Index.BBSBackend.GetAllSCIDInvokeDetailsBySigner(p.SCID, p.Signer)
	}

	result.Invokes = invokes

	return
}

// GetGetInfoDetails
type GetGetInfoDetails_Result struct {
	GetInfoDetails *structures.GetInfo `json:"getInfoDetails"`
}

// GetInfoDetails gets simple getinfo polling Gnomon's gravdb
func GetGetInfoDetails(ctx context.Context) (result GetGetInfoDetails_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var details *structures.GetInfo

	switch gnomon.Index.DBType {
	case "gravdb":
		details = gnomon.Index.GravDBBackend.GetGetInfoDetails()
	case "boltdb":
		details = gnomon.Index.BBSBackend.GetGetInfoDetails()
	}

	result.GetInfoDetails = details

	return
}

// GetSCIDVariableDetails
type (
	GetSCIDVariableDetails_Params struct {
		SCID   string `json:"scid"`
		Height int64  `json:"height"`
	}

	GetAllSCIDVariableDetails_Result struct {
		AllVariables []*structures.SCIDVariable `json:"allVariables"`
	}
)

// GetSCIDVariableDetailsAtTopoheight from Gnomon's gravdb
func GetSCIDVariableDetailsAtTopoheight(ctx context.Context, p GetSCIDVariableDetails_Params) (result GetAllSCIDVariableDetails_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var vars []*structures.SCIDVariable

	switch gnomon.Index.DBType {
	case "gravdb":
		vars = gnomon.Index.GravDBBackend.GetSCIDVariableDetailsAtTopoheight(p.SCID, p.Height)
	case "boltdb":
		vars = gnomon.Index.BBSBackend.GetSCIDVariableDetailsAtTopoheight(p.SCID, p.Height)
	}

	result.AllVariables = vars

	return
}

// GetAllSCIDVariableDetails from Gnomon's gravdb
func GetAllSCIDVariableDetails(ctx context.Context, p SCID_Param) (result GetAllSCIDVariableDetails_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var vars []*structures.SCIDVariable
	switch gnomon.Index.DBType {
	case "gravdb":
		vars = gnomon.Index.GravDBBackend.GetAllSCIDVariableDetails(p.SCID)
	case "boltdb":
		vars = gnomon.Index.BBSBackend.GetAllSCIDVariableDetails(p.SCID)
	}

	result.AllVariables = vars

	return
}

// GetSCIDKeysByValue and GetSCIDValuesByKey
type (
	GetSCIDKeysOrValue_Params struct {
		SCID   string      `json:"scid"`
		Height int64       `json:"height"`
		Value  interface{} `json:"value"` // used for key as well
	}

	GetSCIDKeys_Result struct {
		StringKeys []string `json:"stringKeys"`
		Uint64Keys []uint64 `json:"uint64Keys"`
	}

	GetSCIDValues_Result struct {
		StringValues []string `json:"stringValues"`
		Uint64Values []uint64 `json:"uint64Values"`
	}
)

// GetSCIDKeysByValue at height from Gnomon's gravdb, height 0 will use LastIndexedHeight
func GetSCIDKeysByValue(ctx context.Context, p GetSCIDKeysOrValue_Params) (result GetSCIDKeys_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	if p.Height == 0 {
		p.Height = gnomon.Index.LastIndexedHeight
	}

	var sKeys []string
	var uKeys []uint64

	switch gnomon.Index.DBType {
	case "gravdb":
		sKeys, uKeys = gnomon.Index.GravDBBackend.GetSCIDKeysByValue(p.SCID, p.Value, p.Height, false)
	case "boltdb":
		sKeys, uKeys = gnomon.Index.BBSBackend.GetSCIDKeysByValue(p.SCID, p.Value, p.Height, false)
	}

	result.StringKeys = sKeys
	result.Uint64Keys = uKeys

	return
}

// GetSCIDValuesByKey at height from Gnomon's gravdb, height 0 will use LastIndexedHeight
func GetSCIDValuesByKey(ctx context.Context, p GetSCIDKeysOrValue_Params) (result GetSCIDValues_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	if p.Height == 0 {
		p.Height = gnomon.Index.LastIndexedHeight
	}

	var sKeys []string
	var uKeys []uint64

	switch gnomon.Index.DBType {
	case "gravdb":
		sKeys, uKeys = gnomon.Index.GravDBBackend.GetSCIDValuesByKey(p.SCID, p.Value, p.Height, false)
	case "boltdb":
		sKeys, uKeys = gnomon.Index.BBSBackend.GetSCIDValuesByKey(p.SCID, p.Value, p.Height, false)
	}

	result.StringValues = sKeys
	result.Uint64Values = uKeys

	return
}

// GetLiveSCIDValuesByKey at height from daemon, height 0 will use LastIndexedHeight
func GetLiveSCIDValuesByKey(ctx context.Context, p GetSCIDKeysOrValue_Params) (result GetSCIDValues_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	if p.Height == 0 {
		p.Height = gnomon.Index.LastIndexedHeight
	}

	var stringValues []string
	var uint64Values []uint64
	var variables []*structures.SCIDVariable
	stringValues, uint64Values, err = gnomon.Index.GetSCIDValuesByKey(variables, p.SCID, p.Value, p.Height)
	if err != nil {
		return
	}

	result.StringValues = stringValues
	result.Uint64Values = uint64Values

	return
}

// GetLiveSCIDKeysByValue at height from daemon, height 0 will use LastIndexedHeight
func GetLiveSCIDKeysByValue(ctx context.Context, p GetSCIDKeysOrValue_Params) (result GetSCIDKeys_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	if p.Height == 0 {
		p.Height = gnomon.Index.LastIndexedHeight
	}

	var stringKeys []string
	var uint64Keys []uint64
	var variables []*structures.SCIDVariable
	stringKeys, uint64Keys, err = gnomon.Index.GetSCIDKeysByValue(variables, p.SCID, p.Value, p.Height)
	if err != nil {
		return
	}

	result.StringKeys = stringKeys
	result.Uint64Keys = uint64Keys

	return
}

// GetSCIDInteractionHeight
type GetSCIDInteractionHeight_Result struct {
	InteractionHeights []int64 `json:"interactionHeights"`
}

// GetSCIDInteractionHeight by scid from Gnomon's gravdb
func GetSCIDInteractionHeight(ctx context.Context, p SCID_Param) (result GetSCIDInteractionHeight_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var heights []int64

	switch gnomon.Index.DBType {
	case "gravdb":
		heights = gnomon.Index.GravDBBackend.GetSCIDInteractionHeight(p.SCID)
	case "boltdb":
		heights = gnomon.Index.BBSBackend.GetSCIDInteractionHeight(p.SCID)
	}

	result.InteractionHeights = heights

	return
}

// GetInteractionIndex
type (
	GetInteractionIndex_Params struct {
		Topoheight int64   `json:"topoheight"`
		Heights    []int64 `json:"heights"`
	}
	GetInteractionIndex_Result struct {
		InteractionIndex int64 `json:"interactionIndex"`
	}
)

// GetInteractionIndex by scid from Gnomon's gravdb
func GetInteractionIndex(ctx context.Context, p GetInteractionIndex_Params) (result GetInteractionIndex_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var height int64

	switch gnomon.Index.DBType {
	case "gravdb":
		height = gnomon.Index.GravDBBackend.GetInteractionIndex(p.Topoheight, p.Heights, false)
	case "boltdb":
		height = gnomon.Index.BBSBackend.GetInteractionIndex(p.Topoheight, p.Heights, false)
	}

	result.InteractionIndex = height

	return
}

// GetInvalidSCIDDeploys
type GetInvalidSCIDDeploys_Result struct {
	InvalidDeploys map[string]uint64 `json:"invalidDeploys"`
}

// GetInvalidSCIDDeploys from Gnomon's gravdb
func GetInvalidSCIDDeploys(ctx context.Context) (result GetInvalidSCIDDeploys_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	invalids := make(map[string]uint64)

	switch gnomon.Index.DBType {
	case "gravdb":
		invalids = gnomon.Index.GravDBBackend.GetInvalidSCIDDeploys()
	case "boltdb":
		invalids = gnomon.Index.BBSBackend.GetInvalidSCIDDeploys()
	}

	result.InvalidDeploys = invalids

	return
}

// GetAllMiniblockDetails
type GetAllMiniblockDetails_Result struct {
	MBLdetails map[string][]*structures.MBLInfo `json:"mblDetails"`
}

// GetAllMiniblockDetails from Gnomon's gravdb
func GetAllMiniblockDetails(ctx context.Context) (result GetAllMiniblockDetails_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	details := make(map[string][]*structures.MBLInfo)

	switch gnomon.Index.DBType {
	case "gravdb":
		details = gnomon.Index.GravDBBackend.GetAllMiniblockDetails()
	case "boltdb":
		details = gnomon.Index.BBSBackend.GetAllMiniblockDetails()
	}

	result.MBLdetails = details

	return
}

// GetMiniblockDetailsByHash
type (
	GetMiniblockDetailsByHash_Params struct {
		Blid string `json:"blid"`
	}

	GetMiniblockDetailsByHash_Result struct {
		MBdetails []*structures.MBLInfo `json:"mbDetails"`
	}
)

// GetMiniblockDetailsByHash from Gnomon's gravdb
func GetMiniblockDetailsByHash(ctx context.Context, p GetMiniblockDetailsByHash_Params) (result GetMiniblockDetailsByHash_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var details []*structures.MBLInfo

	switch gnomon.Index.DBType {
	case "gravdb":
		details = gnomon.Index.GravDBBackend.GetMiniblockDetailsByHash(p.Blid)
	case "boltdb":
		details = gnomon.Index.BBSBackend.GetMiniblockDetailsByHash(p.Blid)
	}

	result.MBdetails = details

	return
}

// GetMiniblockCountByAddress
type GetMiniblockCountByAddress_Result struct {
	Miniblocks int64 `json:"mbCount"`
}

// GetMiniblockCountByAddress from Gnomon's gravdb
func GetMiniblockCountByAddress(ctx context.Context, p Address_Param) (result GetMiniblockCountByAddress_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var count int64

	switch gnomon.Index.DBType {
	case "gravdb":
		count = gnomon.Index.GravDBBackend.GetMiniblockCountByAddress(p.Address)
	case "boltdb":
		count = gnomon.Index.BBSBackend.GetMiniblockCountByAddress(p.Address)
	}

	result.Miniblocks = count

	return
}

// GetSCIDInteractionByAddr
type GetSCIDInteractionByAddr_Result struct {
	SCIDs []string `json:"interactionByAddr"`
}

// GetSCIDInteractionByAddr from Gnomon's gravdb
func GetSCIDInteractionByAddr(ctx context.Context, p Address_Param) (result GetSCIDInteractionByAddr_Result, err error) {
	if gnomon.Index == nil {
		err = fmt.Errorf("gnomon is not active")
		return
	}

	var scids []string

	switch gnomon.Index.DBType {
	case "gravdb":
		scids = gnomon.Index.GravDBBackend.GetSCIDInteractionByAddr(p.Address)
	case "boltdb":
		scids = gnomon.Index.BBSBackend.GetSCIDInteractionByAddr(p.Address)
	}

	result.SCIDs = scids

	return
}
