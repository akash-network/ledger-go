/*******************************************************************************
*   (c) 2018 ZondaX GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
********************************************************************************/

package ledger_cosmos_go

import (
	"errors"
	"fmt"
	"math"

	"github.com/zondax/ledger-go"
)

const (
	userCLA = 0x55

	userINSGetVersion       = 0
	userINSSignSECP256K1    = 2
	userINSGetAddrSecp256k1 = 4

	userMessageChunkSize = 250
)

var (
	errLedgerApduBadKeyHandle = "[APDU_CODE_BAD_KEY_HANDLE] The parameters in the data field are incorrect"
	errLedgerApduDataInvalid  = "[APDU_CODE_DATA_INVALID] Referenced data reversibly blocked (invalidated)"
)

var (
	errNotSupportedAppVersion  = errors.New("ledger-cosmos: app version not supported")
	errNotFoundLedgerDevice    = errors.New("ledger-cosmos: couldn't find ledger device")
	errCosmosAppIsNotRunning   = errors.New("ledger-cosmos: are you sure the Cosmos app is open")
	errInvalidResponse         = errors.New("ledger-cosmos: invalid response")
	errNotEnoughTokens         = errors.New("ledger-cosmos: not enough tokens were provided")
	errUnexpectedJsonCharacter = errors.New("ledger-cosmos: unexpected character in JSON string")
	errIncompleteJsonString    = errors.New("ledger-cosmos: the JSON string is not a complete")
)

// LedgerCosmos represents a connection to the Cosmos app in a Ledger Nano S device
type LedgerCosmos struct {
	api     ledger_go.LedgerDevice
	version VersionInfo
}

// FindLedgerCosmosUserApp finds a Cosmos user app running in a ledger device
func FindLedgerCosmosUserApp() (*LedgerCosmos, error) {
	admin := ledger_go.NewLedgerAdmin()
	if admin.CountDevices() == 0 {
		return nil, errNotFoundLedgerDevice
	}

	ledgerAPI, err := admin.Connect(0)

	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = ledgerAPI.Close()
		}
	}()

	app := LedgerCosmos{ledgerAPI, VersionInfo{}}
	appVersion, err := app.GetVersion()

	if err != nil {
		if err.Error() == "[APDU_CODE_CLA_NOT_SUPPORTED] CLA not supported" {
			return nil, errCosmosAppIsNotRunning
		}
		return nil, err
	}

	err = app.CheckVersion(*appVersion)
	if err != nil {
		return nil, err
	}

	return &app, err
}

// Close closes a connection with the Cosmos user app
func (ledger *LedgerCosmos) Close() error {
	return ledger.api.Close()
}

// CheckVersion returns true if the App version is supported by this library
func (ledger *LedgerCosmos) CheckVersion(ver VersionInfo) error {
	version, err := ledger.GetVersion()
	if err != nil {
		return err
	}

	switch version.Major {
	case 1:
		return CheckVersion(ver, VersionInfo{0, 1, 5, 1})
	case 2:
		return CheckVersion(ver, VersionInfo{0, 2, 1, 0})
	default:
		return errNotSupportedAppVersion
	}
}

// GetVersion returns the current version of the Cosmos user app
func (ledger *LedgerCosmos) GetVersion() (*VersionInfo, error) {
	message := []byte{userCLA, userINSGetVersion, 0, 0, 0}
	response, err := ledger.api.Exchange(message)

	if err != nil {
		return nil, err
	}

	if len(response) < 4 {
		return nil, errInvalidResponse
	}

	ledger.version = VersionInfo{
		AppMode: response[0],
		Major:   response[1],
		Minor:   response[2],
		Patch:   response[3],
	}

	return &ledger.version, nil
}

// SignSECP256K1 signs a transaction using Cosmos user app
// this command requires user confirmation in the device
func (ledger *LedgerCosmos) SignSECP256K1(bip32Path []uint32, transaction []byte) ([]byte, error) {
	switch ledger.version.Major {
	case 1:
		return ledger.signv1(bip32Path, transaction)
	case 2:
		return ledger.signv2(bip32Path, transaction)
	default:
		return nil, errNotSupportedAppVersion
	}
}

// GetPublicKeySECP256K1 retrieves the public key for the corresponding bip32 derivation path (compressed)
// this command DOES NOT require user confirmation in the device
func (ledger *LedgerCosmos) GetPublicKeySECP256K1(bip32Path []uint32) ([]byte, error) {
	pubkey, _, err := ledger.getAddressPubKeySECP256K1(bip32Path, "cosmos", false)
	return pubkey, err
}

func validHRPByte(b byte) bool {
	// https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki
	return b >= 33 && b <= 126
}

// GetAddressPubKeySECP256K1 returns the pubkey (compressed) and address (bech(
// this command requires user confirmation in the device
func (ledger *LedgerCosmos) GetAddressPubKeySECP256K1(bip32Path []uint32, hrp string) (pubkey []byte, addr string, err error) {
	return ledger.getAddressPubKeySECP256K1(bip32Path, hrp, true)
}

func (ledger *LedgerCosmos) GetBip32bytes(bip32Path []uint32, _ int) ([]byte, error) {
	var pathBytes []byte
	var err error

	switch ledger.version.Major {
	case 1:
		pathBytes, err = GetBip32bytesv1(bip32Path, 3)
		if err != nil {
			return nil, err
		}
	case 2:
		pathBytes, err = GetBip32bytesv2(bip32Path, 3)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errNotSupportedAppVersion
	}

	return pathBytes, nil
}

func (ledger *LedgerCosmos) signv1(bip32Path []uint32, transaction []byte) ([]byte, error) {
	var packetIndex byte = 1
	var packetCount = 1 + byte(math.Ceil(float64(len(transaction))/float64(userMessageChunkSize)))

	var finalResponse []byte

	var message []byte

	for packetIndex <= packetCount {
		chunk := userMessageChunkSize
		if packetIndex == 1 {
			pathBytes, err := ledger.GetBip32bytes(bip32Path, 3)
			if err != nil {
				return nil, err
			}
			header := []byte{userCLA, userINSSignSECP256K1, packetIndex, packetCount, byte(len(pathBytes))}
			message = append(header, pathBytes...)
		} else {
			if len(transaction) < userMessageChunkSize {
				chunk = len(transaction)
			}
			header := []byte{userCLA, userINSSignSECP256K1, packetIndex, packetCount, byte(chunk)}
			message = append(header, transaction[:chunk]...)
		}

		response, err := ledger.api.Exchange(message)
		if err != nil {
			return nil, processErrorResponse(response, err)
		}

		finalResponse = response
		if packetIndex > 1 {
			transaction = transaction[chunk:]
		}
		packetIndex++

	}
	return finalResponse, nil
}

func (ledger *LedgerCosmos) signv2(bip32Path []uint32, transaction []byte) ([]byte, error) {
	var packetIndex byte = 1
	var packetCount = 1 + byte(math.Ceil(float64(len(transaction))/float64(userMessageChunkSize)))

	var finalResponse []byte

	var message []byte

	for packetIndex <= packetCount {
		chunk := userMessageChunkSize
		if packetIndex == 1 {
			pathBytes, err := ledger.GetBip32bytes(bip32Path, 3)
			if err != nil {
				return nil, err
			}
			header := []byte{userCLA, userINSSignSECP256K1, 0, 0, byte(len(pathBytes))}
			message = append(header, pathBytes...)
		} else {
			if len(transaction) < userMessageChunkSize {
				chunk = len(transaction)
			}

			payloadDesc := byte(1)
			if packetIndex == packetCount {
				payloadDesc = byte(2)
			}

			header := []byte{userCLA, userINSSignSECP256K1, payloadDesc, 0, byte(chunk)}
			message = append(header, transaction[:chunk]...)
		}

		response, err := ledger.api.Exchange(message)
		if err != nil {
			return nil, processErrorResponse(response, err)
		}

		finalResponse = response
		if packetIndex > 1 {
			transaction = transaction[chunk:]
		}
		packetIndex++

	}
	return finalResponse, nil
}

// GetAddressPubKeySECP256K1 returns the pubkey (compressed) and address (bech(
// this command requires user confirmation in the device
func (ledger *LedgerCosmos) getAddressPubKeySECP256K1(bip32Path []uint32, hrp string, requireConfirmation bool) (pubkey []byte, addr string, err error) {
	if len(hrp) > 83 {
		return nil, "", errors.New("ledger-cosmos: hrp len should be <10")
	}

	hrpBytes := []byte(hrp)
	for _, b := range hrpBytes {
		if !validHRPByte(b) {
			return nil, "", errors.New("ledger-cosmos: all characters in the HRP must be in the [33, 126] range")
		}
	}

	pathBytes, err := ledger.GetBip32bytes(bip32Path, 3)
	if err != nil {
		return nil, "", err
	}

	p1 := byte(0)
	if requireConfirmation {
		p1 = byte(1)
	}

	// Prepare message
	header := []byte{userCLA, userINSGetAddrSecp256k1, p1, 0, 0}
	message := append(header, byte(len(hrpBytes)))
	message = append(message, hrpBytes...)
	message = append(message, pathBytes...)
	message[4] = byte(len(message) - len(header)) // update length

	response, err := ledger.api.Exchange(message)

	if err != nil {
		return nil, "", err
	}
	if len(response) < 35+len(hrp) {
		return nil, "", errors.New("ledger-cosmos: invalid response")
	}

	pubkey = response[0:33]
	addr = string(response[33:])

	return pubkey, addr, err
}

func processErrorResponse(response []byte, err error) error {
	// In this special case, we can extract additional info
	msg := string(response)

	switch err.Error() {
	case errLedgerApduBadKeyHandle:
		switch msg {
		case "ERROR: JSMN_ERROR_NOMEM":
			err = errNotEnoughTokens
		case "PARSER ERROR: JSMN_ERROR_INVAL":
			err = errUnexpectedJsonCharacter
		case "PARSER ERROR: JSMN_ERROR_PART":
			err = errIncompleteJsonString
		default:
			err = fmt.Errorf("ledger-cosmos: unknown error - %s", msg)
		}
	case errLedgerApduDataInvalid:
		err = fmt.Errorf("ledger-cosmos: unknown error - %s", msg)
	}

	return err
}
