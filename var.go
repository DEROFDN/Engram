// Copyright 2023-2024 DERO Foundation. All rights reserved.
// Use of this source code in any form is governed by RESEARCH license.
// license can be found in the LICENSE file.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY
// EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL
// THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
// PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
// STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF
// THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package main

import (
	"fyne.io/fyne/v2"
	"github.com/blang/semver"
)

// Globals
var (
	version    = semver.MustParse("0.5.1")
	a          fyne.App
	engram     Engram
	session    Session
	gnomon     Gnomon
	messages   Messages
	status     Status
	tx         Transfers
	res        Res
	colors     Colors
	cyberdeck  Cyberdeck
	themes     Theme
	rpc_client Client
	Connected  bool
	nav        Navigation
	ui         UI
)

// Functions
var (

	// int
	base10 = 10

	// uint64
	destPort = uint64(1337)

	// bytes
	byte0         = []byte("0")
	byte1         = []byte("1")
	byteauthmode  = []byte("auth_mode")
	byteendpoint  = []byte("endpoint")
	bytefalse     = []byte("false")
	bytegnomon    = []byte("gnomon")
	byteLastScan  = []byte("Last Scan")
	bytemode      = []byte("mode")
	bytenetwork   = []byte("network")
	bytetrue      = []byte("true")
	byteusername  = []byte("username")
	byteusernames = []byte("usernames")

	// ""
	string_ = ""

	// newline
	singlenewline = "\n"

	// int

	// " "
	singlespace          = " "
	singlebackslash      = " / "
	centeredthreeperiods = " ... "
	balanceBanner        = " B A L A N C E "
	mainnetBanner        = " M  A  I  N  N  E  T "
	modulesBanner        = " M O D U L E S "
	moreOptionsBanner    = " M O R E   O P T I O N S "
	optionalBanner       = " O P T I O N A L "
	regBanner            = " R E G I S T R A T I O N "
	statusBanner         = " S T A T U S "
	testnetBanner        = " T  E  S  T  N  E  T "
	numbersOnly          = " (Numbers Only)"
	centeredCreate       = " Create "
	offlineMode          = " Offline Mode"
	centeredRegister     = " Register "
	centeredSend         = " Send "
	showOnlyRecent       = " Show only recent messages"

	// "  "
	doublespace             = "  "
	zero                    = "  0"
	plus                    = "  + "
	subtract                = "  - "
	atomicZeroes            = "  0.00000"
	transferBanner          = "  T R A N S F E R  "
	disabledOffline         = "  Asset Explorer is disabled in offline mode."
	disabledTrackingOffline = "  Asset tracking is disabled in offline mode."
	disabledGnomon          = "  Asset Explorer is disabled. Gnomon is inactive."
	disabledTrackingGnomon  = "  Asset tracking is disabled. Gnomon is inactive."
	gatheringSCIDs          = "  Gathering an index of smart contracts... "
	gnomonCenter            = "  Gnomon is syncing... [%d / %d]"
	indexingResults         = "  Indexing Gnomon results... Please wait."
	indexingComplete        = "  Indexing complete - Scanning balances..."
	loadingHistory          = "  Loading previous scan history..."
	loadingScan             = "  Loading previous scan results..."
	centeredOwned           = "  Owned Assets:  %d"
	centeredOwnedScanned    = "  Owned Assets:  %d  ( %s )"
	received                = "  Received"
	centeredResults         = "  Results:  %d"
	scanning                = "  Scanning..."
	scanningSCIDs           = "  Scanning smart contracts... "
	centeredSearch          = "  Search History:  %d"
	senderAddress           = "  SENDER  ADDRESS"
	sentRight               = "  Sent"
	txHistroy               = "  Transaction History"

	// "   "
	threespaces     = "   "
	amount          = "   AMOUNT"
	blockHeight     = "   BLOCK  HEIGHT"
	assetBALANCE    = "   ASSET  BALANCE"
	destinationPort = "   DESTINATION  PORT"
	executeACTION   = "   EXECUTE  ACTION"
	payLoad         = "   PAYLOAD"
	payDirection    = "   PAYMENT  DIRECTION"
	receiverAddress = "   RECEIVER  ADDRESS"
	replyAddress    = "   REPLY  ADDRESS"
	scAUTHOR        = "   SMART  CONTRACT  AUTHOR"
	scOWNER         = "   SMART  CONTRACT  OWNER"
	scID            = "   SMART  CONTRACT  ID"
	serviceAddress  = "   SERVICE  ADDRESS"
	sourcePort      = "   SOURCE  PORT"
	txFees          = "   TRANSACTION  FEES"
	transferASSET   = "   TRANSFER  ASSET"
	txID            = "   TRANSACTION  ID"
	txProof         = "   TRANSACTION  PROOF"

	// "     "
	txDetail = "     Transaction Detail"

	// period
	doubleperiod = ".."
	threeperiods = "..."

	// tilde
	threetilde = "~~~"

	// dots
	fourdots = "····"

	// #
	numberOfBlocks  = "# of Latest Blocks (Optional)"
	nameNotProvided = "## No name provided"
	doublehashes    = "## "
	header3Service  = "### SERVICE"
	header3Normal   = "### NORMAL"

	// (
	selectAccount   = "(Select Account)"
	selectLanguage  = "(Select Language)"
	selectOne       = "(Select One)"
	selectTxType    = "(Select Transaction Type)"
	selectAnonymity = "(Select Anonymity Set)"

	// [
	boxAmount                     = "[%s] Amount: %d\n"
	errBuildTx                    = "[%s] Build Transaction Error: %s\n"
	boxErr                        = "[%s] Err: %s\n"
	successFuncTx                 = "[%s] Function execution successful - TXID:  %s\n"
	errGasEstimate                = "[%s] GasEstimate Error: %s\n"
	errSendTx                     = "[%s] Send Tx Error: %s"
	boxString                     = "[%s] String: %s\n"
	successSendTx                 = "[%s] Username transfer successful - TXID:  %s\n"
	errorSavingSearch             = "[Asset Explorer] Error saving search result: %s\n"
	errStoring                    = "[Asset] Error storing new asset balance for: %s\n"
	fountAssets                   = "[Assets] Found asset: %s\n"
	errDeletingDatapad            = "[Datapad] Error deleting %s: %s\n"
	errDatashard                  = "[Datapad] Err: %s\n"
	notExported                   = "[DVM] Function %s is not an exported function - skipping it\n"
	isInit                        = "[DVM] Function %s is an initialization function - skipping it\n"
	engramMsg                     = "[Engram] %s\n"
	cyberdeckShuttingdown         = "[Engram] Cyberdeck closed.\n"
	errPurgeData                  = "[Engram] Error purging local datashard data: %s\n"
	successPurgeData              = "[Engram] Local datashard data has been purged successfully\n"
	rpcShuttingdown               = "[Engram] RPC client closed.\n"
	scanTracking                  = "[Engram] Scan tracking enabled, only scanning the last %d blocks...\n"
	setMinRingSize                = "[Engram] Set minimum ring size: %d\n"
	engramShuttingdown            = "[Engram] Shutting down wallet services...\n"
	valueUpdated                  = "[Engram] Value conversion updated.\n"
	shutdownSucces                = "[Engram] Wallet saved and closed successfully.\n"
	websocketShuttingdown         = "[Engram] Websocket client closed.\n"
	noWallet                      = "[error] no wallet found.\n"
	failTxData                    = "[getTxData] TXID: %s (Failed: %s)\n"
	gnomonAssetScan               = "[Gnomon] Asset Scan Status: [%d / %d / %d]\n"
	gnomonClosed                  = "[Gnomon] Closed all indexers.\n"
	errGnomonPurge                = "[Gnomon] Error purging local Gnomon data: %s\n"
	successGnomonPurge            = "[Gnomon] Local Gnomon data has been purged successfully\n"
	gnomonFailure                 = "[Gnomon] Querying usernames failed: %s\n"
	gnomonScan                    = "[Gnomon] Scan Status: [%d / %d]\n"
	gnomonShuttingdown            = "[Gnomon] Shutting down indexers...\n"
	errFailedStore                = "[History] Failed to store asset: %s\n"
	successDispatch               = "[Message] Dispatched transaction successfully to: %s\n"
	failedToSend                  = "[Message] Failed to send: %s\n"
	dispatchedTx                  = "[Message] Dispatched transaction: %s\n"
	errDispatching                = "[Message] Error while dispatching transaction: %s\n"
	errBuilding                   = "[Message] Error while building transaction: %s\n"
	calcFees                      = "[Message] Calculated Fees: %d\n"
	packErr                       = "[Message] Arguments packing err: %s\n"
	checkingRings                 = "[Messages] Checking ring members for TXID: %s (Failed: %s)\n"
	checkingRingsUnverified       = "[Messages] Checking ring members for TXID: %s (Unverified - Skipping)\n"
	checkingRingsVerified         = "[Messages] Checking ring members for TXID: %s (Verified)\n"
	attemptConnection             = "[Network] Attempting network connection to: %s\n"
	failedDaemonConnection        = "[Network] Could not connect to daemon...%d\n"
	failedConnection              = "[Network] Failed to connect to: %s\n"
	networkOffline                = "[Network] Offline › Last Height: "
	regStartNotice                = "[Registration] Account registration PoW started...\n"
	regErr                        = "[Registration] Error: %s\n"
	regInprogressNotice           = "[Registration] Registering your account. This can take up to 120 minutes (one time). Please wait...\n"
	regSuccess                    = "[Registration] Registration transaction dispatched successfully.\n"
	regTXID                       = "[Registration] Registration TXID: %s\n"
	addArgs                       = "[Send] Added arguments..\n"
	addedTransfer                 = "[Send] Added transfer to the pending list.\n"
	sentAmount                    = "[Send] Amount: %d\n"
	argPackErr                    = "[Send] Arguments packing err: %s\n"
	sendBalance                   = "[Send] Balance: %d\n"
	checkingAmount                = "[Send] Checking Amount..\n"
	checkingArgs                  = "[Send] Checking arguments..\n"
	checkPack                     = "[Send] Checking Pack..\n"
	checkingPayid                 = "[Send] Checking payment ID/destination port..\n"
	checkingServices              = "[Send] Checking services..\n"
	dispatchTx                    = "[Send] Dispatched transaction: %s\n"
	dispatchErr                   = "[Send] Error while dispatching transaction: %s\n"
	sendError                     = "[Send] Error: %s\n"
	invalidRings                  = "[Send] Error: Invalid ringsize - New ringsize = %d\n"
	sentInsufficentFunds          = "[Send] Error: Insufficient funds"
	notIntegrated                 = "[Send] Not Integrated..\n"
	ringSize                      = "[Send] Ringsize: %d\n"
	startTx                       = "[Send] Starting tx...\n"
	sendErr                       = "[Send Message] Error: %s\n"
	integratedAddresWithDest      = "[Send Message] Destination port is integrated in address. %x\n"
	integratedAddresWithoutDest   = "[Send Message] Integrated Address does not contain destination port.\n"
	integratedMessageIs           = "[Send Message] Integrated Message: %s\n"
	addressDoesExpireOn           = "[Send Message] This address will expire on %x\n"
	addressExpiredOn              = "[Send Message] This address has expired on %x\n"
	destIntegrated                = "[Service] Destination port is integrated in address."
	integratedMessage             = "[Service] Integrated Message"
	integratedAddressCantValidate = "[Service] Integrated Address arguments could not be validated"
	noDestPort                    = "[Service] Integrated Address does not contain destination port"
	replyRequired                 = "[Service] Reply Address required, sending: %s\n"
	addressWillExpire             = "[Service] This address will expire "
	addressExpired                = "[Service] This address has expired."
	timeUnsupported               = "[Service] Time currently not supported.\n"
	txAmount                      = "[Service] Transaction amount: %s\n"
	integratedArgs                = "[Service Address] Arguments: %s\n"
	errService                    = "[Service Address] Error: %s\n"
	newIntegrated                 = "[Service Address] New Integrated Address: %s\n"
	failedParse                   = "[Transfer] Failed parsing transfer amount: %s\n"
	failedBuild                   = "[Transfer] Failed to build transaction: %s\n"
	failedSend                    = "[Transfer] Failed to send asset: %s - %s\n"
	successSend                   = "[Transfer] Successfully sent asset: %s - TXID: %s\n"
	successTransfer               = "[TransferOwnership] %s was successfully transfered to: %s\n"
	failTransfer                  = "[TransferOwnership] %s was unsuccessful in transferring to: %s\n"
	userMsg                       = "[Username] %s\n"
	errQuerying                   = "[Username] Error querying usernames: %s\n"
	usrError                      = "[Username] Error: %s\n"
	userErr                       = "[username] error: skipping registration - username exists.\n"
	successReg                    = "[Username] Successfully registered username: %s\n"
	userRegTxID                   = "[Username] Username Registration TXID:  %s\n"

	// comma
	singlecoma = ","

	// colon
	singlecolon = ":"

	// asterisks
	singleasterisks = "*"

	// semicolons
	threesemicolons = ";;;"
	foursemicolons  = ";;;;"
	sixsemicolons   = ";;;;;;"

	// dash
	valueBlank                   = "-.--"
	doubledashes                 = "--"
	stringFlagallowrpcpasschange = "--allow-rpc-password-change"
	stringFlagdaemonaddress      = "--daemon-address"
	stringFlagdebug              = "--debug"
	stringFlageoffline           = "--offline"
	stringFlagp2pbind            = "--p2p-bind"
	stringFlagrpcbind            = "--rpc-bind"
	stringFlagrpclogin           = "--rpc-login"
	stringFlagrpcserver          = "--rpc-server"
	stringFlagremote             = "--remote"
	stringFlagtestnet            = "--testnet"
	threedashes                  = "---"
	integratedBanner             = "-------------    INTEGRATED  ADDRESS    -------------"

	// 0-9
	string1    = "1"
	dateFormat = "2006-01-02"

	//0.0
	onederi = "0.00001"
	twoderi = "0.00002"

	// app details

	accountrecoveryBanner = "A C C O U N T    R E C O V E R Y"
	cyberdeckBanner       = "C Y B E R D E C K"
	datapadBanner         = "D A T A P A D"
	errorBanner           = "E  R  R  O  R"
	historyBanner         = "H I S T O R Y"
	identityBanner        = "I D E N T I T Y"
	identitydetailBanner  = "I D E N T I T Y    D E T A I L"
	messagesBanner        = "M E S S A G E S"
	myaccountBanner       = "M Y    A C C O U N T"
	newpassBanner         = "N E W    P A S S W O R D"
	txdetailBanner        = "T R A N S A C T I O N    D E T A I L"
	transferdetailBanner  = "T R A N S F E R    D E T A I L"
	transfersBanner       = "T R A N S F E R S"

	// a
	usrpassMsg = "A username and password is required in order to allow application connectivity."

	// access
	accessRecovery = "Access Recovery Words"

	// account
	accountNotCreated   = "Account could not be created."
	accountName         = "Account Name"
	accountTooLong      = "Account name is too long."
	accountExists       = "Account name already exists."
	accountLength       = "Account name is too long (max 30 characters)."
	accountCreated      = "Account successfully created."
	accountVerification = "ACCOUNT  VERIFICATION  REQUIRED"

	// add
	addTxDetails = "Add Transfer Details"

	// address
	addressDoesntExist = "address does not exist"

	// allow
	stringAllowed = "Allowed"

	//amount
	stringAmount = "Amount"

	// anon
	anonSetNone        = "Anonymity Set:   2  (None)"
	anonSetLower       = "Anonymity Set:   4  (Low)"
	anonSetLess        = "Anonymity Set:   8  (Low)"
	anonSetRecommended = "Anonymity Set:   16  (Recommended)"
	anonSetMore        = "Anonymity Set:   32  (Medium)"
	anonSetHigh        = "Anonymity Set:   64  (High)"
	anonSetMost        = "Anonymity Set:   128  (High)"

	// app
	appCreate          = "app.create"
	appCyberdeck       = "app.cyberdeck"
	appDatapad         = "app.datapad"
	appExplorer        = "app.explorer"
	appIdentity        = "app.Identity"
	appMain            = "app.main"
	appMainLoading     = "app.main.loading"
	appManager         = "app.manager"
	appMessages        = "app.messages"
	appMessagesContact = "app.messages.contact"
	appRegister        = "app.register"
	appRestore         = "app.restore"
	appSend            = "app.send"
	appService         = "app.service"
	appSettings        = "app.settings"
	appTransfers       = "app.transfers"
	appWallet          = "app.wallet"
	appConnections     = "APPLICATION  CONNECTIONS"

	// are
	confirmDelete = "Are you sure?"

	// assets
	assetsManager = "Asset Manager"
	assetAmount   = "Asset Amount (Numbers Only)"
	assetScan     = "Asset Scan"
	assetExplorer = "Asset Explorer"
	assetVALUE    = "ASSETVALUE"

	// navigation

	// back
	backtoExplorer  = "Back to Asset Explorer"
	backtoDash      = "Back to Dashboard"
	backtoDatapad   = "Back to Datapad"
	backtoHistory   = "Back to History"
	backtoIdentity  = "Back to Identity"
	backtoMessages  = "Back to Messages"
	backtoAccount   = "Back to My Account"
	backtoTransfers = "Back to Transfers"

	// bal
	formatBalance = "Balance:  "

	// block
	stringBlocked = "Blocked"

	// cancel
	stringCancel   = "Cancel"
	cancelTransfer = "Cancel Transfer"

	// clear
	stringClear  = "Clear"
	clearData    = "Clear Local Data"
	clearDatapad = "Clear Datapad?"

	// close
	stringClose = "Close"

	// coin
	stringCoinbase = "Coinbase"

	// comment
	stringcomment = "comment"

	// confirm
	txConfirming           = "Confirming..."
	txConfirmingSomeofSome = "Confirming... (%d/%d)"
	confirmPassword        = "Confirm Password"

	// connect
	connectFailure   = "Connection Failure"
	stringCONNECTION = "CONNECTION"

	// contents
	stringContents = "Contents"

	// copy
	copyAddress        = "Copy Address"
	copyCreds          = "Copy Credentials"
	copyPayload        = "Copy Payload"
	copyRecovery       = "Copy Recovery Words"
	copySCID           = "Copy SCID"
	copyServiceAddress = "Copy Service Address"
	copyTxId           = "Copy Transaction ID"
	copyTxProof        = "Copy Transaction Proof"

	// copyright
	copyrightNoticeVersion = "© 2023  DERO FOUNDATION  |  VERSION  "
	copyrightNotice        = "Copyright 2023-2024 DERO Foundation. All rights reserved.\n"

	// could
	writeFailMsg = "Could not write data to disk, please check to make sure Engram has the proper permissions."

	// create
	stringCreate         = "Create"
	createAccount        = "Create a new account"
	createServiceAddress = "CREATE  SERVICE  ADDRESS"

	// current
	currentPass = "Current Password"

	// cyber
	stringCyberdeck = "Cyberdeck"

	// daemon
	stringdaemon = "daemon"

	// data
	stringDatapad  = "Datapad"
	datapadName    = "Datapad Name"
	datapadExists  = "Datapad already exists"
	datapadReset   = "DATAPAD  RESET  REQUESTED"
	datapadChanged = "DATAPAD  CHANGE  DETECTED"
	stringDatapads = "Datapads"

	// datashard
	stringdatashards = "datashards"
	datashardDelete  = "DATASHARD  DELETION  REQUESTED"

	// delete
	stringDelete    = "Delete"
	deleteDatashard = "Delete Datashard"
	deleteDatapad   = "Delete Datapad?"
	deleteSuccess   = "Deletion successful!"

	// dero
	deroAmountNumbers = "DERO Amount (Numbers Only)"
	deroVALUE         = "DEROVALUE"

	// dest
	destinationIntegrated = "Destination port is integrated in address. %d\n"

	// disable
	disabledinOfflineMode = "Disabled in Offline Mode"
	disabledOfflineMode   = "Disabled in Offline Mode"

	// discard
	discardDatapad = "Discard Changes"

	// dst
	dstport = "dst port"

	// empty
	emptyPaymentID = "Empty Payment ID"

	// engram
	appName    = "Engram"
	engramBeta = "Engram v%s (Beta)\n"

	// enable
	enableGnomon = "Enable Gnomon"

	// english
	stringenglish = "english"

	// enter
	scanBlocks = "Enter the number of past blocks that the wallet should scan:"

	// err
	stringError          = "Error"
	erroPassword         = "Error changing password"
	errorCreatingDatapad = "Error creating new Datapad"
	errorDelete          = "Error deleting datashard"
	errorExecuting       = "Error executing function..."
	errSCVars            = "Error getting SC variables: %s\n"
	errorSaving          = "Error saving Datapad"
	errorParsingAssetBal = "error parsing asset balance"
	errGeneral           = "Error: %s\n"
	errPriceQuote        = "error: could not query price from coingecko"
	errGravitonKey       = "error: missing graviton key input"
	errGravitionTree     = "error: missing graviton tree input"
	errNoActive          = "error: no active account found"

	// execute
	stringExecute           = "Execute"
	executeCONTRACTFUNCTION = "EXECUTE  CONTRACT  FUNCTION"
	executing               = "Executing..."

	// explore
	explorerHistory = "Explorer History"

	// export
	exportPlainText = "Export (Plaintext)"

	// expiry
	expirytime = "expiry time"

	// f

	// failed
	failttoSend    = "Failed to send message..."
	failedtoVerify = "Failed to verify address..."

	// function
	executionSuccess = "Function executed successfully!"
	functionInit     = "Function Initialize"

	// grav
	stringgravdb = "gravdb"

	// gnomon
	stringgnomon        = "gnomon"
	stringGNOMON        = "GNOMON"
	stringgnomonTestnet = "gnomon_testnet"
	gnomongMsg          = "Gnomon scans and indexes blockchain data in order to unlock more features, like native asset tracking."
	gnomonDeleted       = "Gnomon data successfully deleted."

	// go
	goBack = "Go Back"

	// http
	endpointInstallSC = "http://%s:%d/install_sc"

	// https
	priceURL = "https://api.coingecko.com/api/v3/simple/price?ids=dero&vs_currencies=usd"

	// ID
	id10tError = "ID-10T Error Protocol"

	// io
	appID = "io.dero.engram"

	// incorrect
	passIncorrect = "Incorrect password entered"

	// id
	stringIdentity = "Identity"

	// init
	stringInitialize        = "Initialize"
	stringInitializePrivate = "InitializePrivate"

	// integrated
	integratedMsg = "Integrated Message: %s\n"

	// insist
	quote = "\"Insist on yourself; never imitate. Your own gift you can present every moment with the \n" +
		"cumulative force of a whole life's cultivation; but of the adopted talent of another, \n" +
		"you have only an extemporaneous, half possession.\"\n\n"

	// insufficient
	insufficientAssetBal = "insufficient asset balance"
	insufficientFunds    = "Insufficient funds"
	insufficientTxAmount = "Insufficient transfer amount..."

	// invalid
	invalidAddress  = "Invalid Address"
	invalidAmount   = "invalid amount entered"
	invalidHost     = "invalid host name"
	invalidPassword = "Invalid password..."
	invalidSeedWord = "Invalid seed word"
	invalidContact  = "invalid username or address"
	invalidTxAmount = "Invalid transaction amount"
	invalidRingsize = "Invalid ringsize"

	// last
	lastUpdated = "Last Updated:  "

	// main
	stringmain    = "main"
	stringmainnet = "mainnet"
	stringMainnet = "Mainnet"

	// message
	stringMessage  = "Message"
	msgAuthor      = "Message the Author"
	msgOwner       = "Message the Owner"
	messageTooLong = "Message too long..."
	msgTooLong     = "Message too long"
	stringMessages = "Messages"

	// my
	myAccount  = "My Account"
	myAssets   = "My Assets"
	myContacts = "My Contacts"
	mySettings = "My Settings"

	// N
	notAvaliable = "N/A"

	// name
	stringname = "name"

	// new
	newAccount        = "New Account"
	newMessage        = "New Message"
	newPassword       = "New Password"
	newServiceAddress = "New Service Address"
	newUser           = "New Username"

	// net
	stringNETWORK = "NETWORK"

	// no
	noAccountSelected = "No account selected..."
	noDescription     = "No description provided"

	// normal
	stringNormal = "Normal"

	// offline
	stringOffline = "Offline"
	stringOFFLINE = "OFFLINE"

	// offset
	offsetFloat = "offset: %f\n"

	// ok
	stringOK = "OK"

	// os
	osArchGoMax = "OS: %s ARCH: %s GOMAXPROCS: %d\n\n"

	// pass
	stringPassword = "Password"
	passUpdated    = "Password Updated"
	passMismatch   = "Passwords do not match"

	// pay
	payIDSlashServicePort = "Payment ID / Service Port"

	// please
	confirmPass      = "Please confirm your password."
	plsNameDatapad   = "Please enter a Datapad name"
	enterPasword     = "Please enter a password."
	enterAccountName = "Please enter an account name."
	recoveryWarning  = "Please save the following 25 recovery words in a safe place. These are the keys to your account, so never share them with anyone."
	enterLanguage    = "Please select a language."
	plsWaitMsg       = "Please wait..."

	// primary
	primaryUser = "PRIMARY  USERNAME"

	// proof
	proofofWork = "PROOF-OF-WORK"

	// receive
	strinReceived   = "Received"
	receiverContact = "Receiver username or address"

	// recover
	stringRecover          = "Recover"
	recoverAccount         = "Recover Account"
	recoverAccountExisting = "Recover an existing account"
	stringRecovery         = "Recovery"
	recoveryWords          = "Recovery Words"

	// reg
	successfulRegister = "Registration successful!"
	registerUser       = "REGISTERED  USERNAME"

	// rescan
	rescanBlockchain = "Rescan Blockchain"

	// restore
	restoreDefaults = "Restore Defaults"

	// resource
	stringResources = "Resources"

	// return
	returntoLogin = "Return to Login"

	// review
	reviewSettings = "Review Settings"

	// save
	stringSave     = "Save"
	saveDatapad    = "Save Datapad?"
	savedTransfers = "Saved Transfers"

	// scroll
	scrollBefore = "scrollBox - before: %f\n"
	scrollAfter  = "scrollBox - after: %f\n"

	// search
	stringSearch  = "Search"
	searchSCID    = "Search By SCID"
	searchContact = "Search for a Contact"

	// security
	stringSECURITY = "SECURITY"

	// seed
	seedWord1  = "Seed Word 1"
	seedWord2  = "Seed Word 2"
	seedWord3  = "Seed Word 3"
	seedWord4  = "Seed Word 4"
	seedWord5  = "Seed Word 5"
	seedWord6  = "Seed Word 6"
	seedWord7  = "Seed Word 7"
	seedWord8  = "Seed Word 8"
	seedWord9  = "Seed Word 9"
	seedWord10 = "Seed Word 10"
	seedWord11 = "Seed Word 11"
	seedWord12 = "Seed Word 12"
	seedWord13 = "Seed Word 13"
	seedWord14 = "Seed Word 14"
	seedWord15 = "Seed Word 15"
	seedWord16 = "Seed Word 16"
	seedWord17 = "Seed Word 17"
	seedWord18 = "Seed Word 18"
	seedWord19 = "Seed Word 19"
	seedWord20 = "Seed Word 20"
	seedWord21 = "Seed Word 21"
	seedWord22 = "Seed Word 22"
	seedWord23 = "Seed Word 23"
	seedWord24 = "Seed Word 24"
	seedWord25 = "Seed Word 25"

	// select
	selectOption = "Select an Option ..."
	selectModule = "Select Module ..."
	selectNode   = "Select Public Node ..."

	// send
	stringSend    = "Send"
	sendAsset     = "Send Asset"
	sendMoney     = "Send Money"
	sendTransfers = "Send Transfers"

	// sent
	stringSent = "Sent"
	sentLeft   = "Sent    "

	// set
	setPrimaryUser    = "Set Primary Username"
	settingTransfer   = "Setting up transfer..."
	settinguptransfer = "Setting up transfer..."
	stringsettings    = "settings"
	stringSettings    = "Settings"

	// service
	stringServices = "Services"

	// sign
	signIn  = "Sign In"
	signOut = "Sign Out"

	// submit
	stringSubmit = "Submit"

	// success
	stringSuccess       = "Success"
	successfullyCreated = "Successfully Created"

	// sys
	systemMalfunction = "System malfunction... Please... Find... Help..."

	// test
	stringtestnet = "testnet"
	stringTestnet = "Testnet"

	// this
	oneTimeMsg = "This one-time process can take a while."

	// transaction
	txFailed = "Transaction Failed..."

	// transfer
	transferSuccessful = "Transfer Successful!"
	transferUser       = "Transfer Username"
	stringTransfers    = "Transfers"

	// true
	stringtrue = "true"

	// trun
	turnOff = "Turn Off"
	turnOn  = "Turn On"

	// unset
	stringUnsent = "Unsent"

	// user
	stringUsername  = "Username"
	userExists      = "Username already exists"
	userDoesntExist = "Username does not exist"
	usertooShort    = "Username too short, need a minimum of six characters"
	usernameAddress = "Username or Address"
	stringUsernames = "Usernames"

	// unable
	unableRegister = "Unable to register..."

	// usd
	usdZero    = "USD  0.00"
	usdWithPad = "USD  "

	// view
	viewHistory  = "View History"
	viewExplorer = "View in Explorer"

	// write
	writeFailure = "Write Failure"

	// you
	dontOwnAsset           = "You do not own this asset"
	accountRecoverySuccess = "Your account has been successfully recovered. "
)
