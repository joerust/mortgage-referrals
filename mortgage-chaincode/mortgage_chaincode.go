/*
Copyright IBM Corp 2016 All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
		 http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
	"strings"
    "encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type CustomerReferral struct {
	ReferralId string `json:"referralId"`
    CustomerName string `json:"customerName"`
	ContactNumber string `json:"contactNumber"`
	CustomerId string `json:"customerId"`
	EmployeeId string `json:"employeeId"`
	Departments []string `json:"departments"`
    CreateDate int64 `json:"createDate"`
	Status string `json:"status"`
	Mortgage Mortgage `json:"mortgage"`
}

type Mortgage struct {
	MortgageNumber string `json:"mortgageNumber"`
    MortgageType string `json:"mortgageType"`
	ReferralId string `json:"referralId"`
	Rate string `json:"rate"`
	Amount string `json:"amount"`
}

// ReferralChaincode implementation stores and updates referral information on the blockchain
type ReferralChaincode struct {
}

func main() {
	err := shim.Start(new(ReferralChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}	
}

func BytesToString(b []byte) string {
    bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
    sh := reflect.StringHeader{bh.Data, bh.Len}
    return *(*string)(unsafe.Pointer(&sh))
}

// Init resets all the things
func (t *ReferralChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	// There is no initialization to do
	return nil, nil
}

// Invoke is our entry point to invoke a chaincode function
func (t *ReferralChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {
		return t.Init(stub, "init", args)
	} else if function == "createReferral" {
		return t.createReferral(stub, args)
	} else if function == "updateReferralStatus" {
		return t.updateReferralStatus(stub, args)
	}
	fmt.Println("invoke did not find func: " + function)

	return nil, errors.New("Received unknown function invocation")
}

// Query is our entry point for queries
func (t *ReferralChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" { //read a variable
		return t.read(stub, args)
	} else if function == "searchByStatus" {
		return t.searchByStatus(args[0], stub)
	} else if function == "searchByDepartment" {
		return t.searchByDepartment(args[0], stub)
	}
	fmt.Println("query did not find func: " + function)

	return nil, errors.New("Received unknown function query")
}

// Adds the referral id to a ledger list item for the given department allowing for quick search of referrals in a given department
func (t *ReferralChaincode) indexByDepartment(referralId string, department string, stub *shim.ChaincodeStub) (error) {
	valAsbytes, err := stub.GetState(department)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + department + "\"}"
		return errors.New(jsonResp)
	}
	
	if valAsbytes == nil {
		err = stub.PutState(department, []byte(referralId))
	} else {
	    commaDelimitedStatuses := BytesToString(valAsbytes)
		err = stub.PutState(department, []byte(commaDelimitedStatuses + "," + referralId))
	}
	
	return err
}

func (t *ReferralChaincode) removeStatusReferralIndex(referralId string, status string, stub *shim.ChaincodeStub) (error) {
	valAsbytes, err := stub.GetState(status)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + status + "\"}"
		return errors.New(jsonResp)
	}
	
	if valAsbytes == nil {
		return nil;
	} else {
		// Remove the referral from this status type, if it exists
		commaDelimitedStatuses := BytesToString(valAsbytes)
		referralIdsInCurrentStatus := strings.Split(commaDelimitedStatuses, ",")
		updatedReferralIdList := ""
		
		appendComma := false
		for i := range referralIdsInCurrentStatus {
			if referralIdsInCurrentStatus[i] != referralId {
			    if appendComma == false {
					updatedReferralIdList += referralIdsInCurrentStatus[i]
					appendComma = true
				} else {
					updatedReferralIdList = updatedReferralIdList + "," + referralIdsInCurrentStatus[i]
				}
			}
		}
		
		err = stub.PutState(status, []byte(updatedReferralIdList))
	}
	
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to update state for " + status + "\"}"
		return errors.New(jsonResp)
	}
	
	return nil
}

// Adds the referral id to a ledger list item for the given department allowing for quick search of referrals in a given department
func (t *ReferralChaincode) indexByStatus(referralId string, status string, stub *shim.ChaincodeStub) (error) {
	valAsbytes, err := stub.GetState(status)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + status + "\"}"
		return errors.New(jsonResp)
	}
	
	if valAsbytes == nil {
		err = stub.PutState(status, []byte(referralId))
	} else {
	    commaDelimitedStatuses := BytesToString(valAsbytes)
		err = stub.PutState(status, []byte(commaDelimitedStatuses + "," + referralId))
	}
	
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to update state for " + status + "\"}"
		return errors.New(jsonResp)
	}
	
	return nil
}

func unmarshallBytes(valAsBytes []byte) (error, CustomerReferral) {
	var err error
	var referral CustomerReferral

	err = json.Unmarshal(valAsBytes, &referral)

	fmt.Println("JSON Unmarshalled")	
	if err != nil {
		fmt.Println(err.Error())
	}
	
	return err, referral
}

func (t *ReferralChaincode) marshallReferral(referral CustomerReferral) (error, []byte) {
	fmt.Println("Marshalling JSON to bytes")
	valAsbytes, err := json.Marshal(referral)
	
	if err != nil {
		fmt.Println("Marshalling JSON to bytes failed")
		return err, nil
	}
	
	return nil, valAsbytes
}

func (t *ReferralChaincode) updateStatus(referral CustomerReferral, status string, stub *shim.ChaincodeStub) (error) {
	fmt.Println("Setting status")
	
	err := t.removeStatusReferralIndex(referral.ReferralId, referral.Status, stub)
	if err != nil {
		return err
	}
	referral.Status = status
	err = t.indexByStatus(referral.ReferralId, status, stub)
	
	return err
}

// updateReferral - invoke function to updateReferral key/value pair
func (t *ReferralChaincode) updateReferralStatus(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var key, value string
	var err error
	var referral CustomerReferral
	var valAsbytes []byte
	
	fmt.Println("running updateReferral()")

	if len(args) != 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4 or more. name of the key and value to set")
	}

	key = args[0] // The referral id
	value = args[1] // The new status
	
	// Look up the json blob that matches the current referral id
	valAsbytes, err = stub.GetState(key)
	
	// Unmarshall said json blob into a referral object
	err = json.Unmarshal(valAsbytes, &referral)
	
	// Save the current status so that it can be unindexed once we update the referral object
	oldStatus := referral.Status;
	
	// Set the referral status to the new value
	referral.Status = value;
	
	// Serialize the object to a JSON string to be stored in the ledger
	valAsbytes, err = json.Marshal(referral)
	
	// Store the json string in the ledger
	err = stub.PutState(key, valAsbytes) //write the variable into the chaincode state
	
	if err != nil {
		return nil, err
	}
	
	// Index things by the new status
	err = t.indexByStatus(key, referral.Status, stub)
	
	if err != nil {
		return []byte("Count not index the bytes by status from the value: " + value + " on the ledger"), err
	}
	
	// Remove the indexing by the status before the update
	err = t.removeStatusReferralIndex(key, oldStatus, stub)
	
	return nil, nil
}

// createReferral - invoke function to write key/value pair
func (t *ReferralChaincode) createReferral(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	var key, value string
	var err error
	fmt.Println("running createReferral()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2 parameters, name of the key and value to set")
	}

	key = args[0] //rename for funsies
	value = args[1]
	
	err = stub.PutState(key, []byte(value)) //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	
	var referral CustomerReferral

	err = json.Unmarshal([] byte(value), &referral)

	
	// Deserialize the input string into a GO data structure to hold the referral
	err = t.indexByStatus(key, referral.Status, stub)
	
	if err != nil {
		return []byte("Count not index the bytes by status from the value: " + value + " on the ledger"), err
	}
	
	// Create a ledger record that indexes the referral id by the created department
	for i := range referral.Departments {
		err = t.indexByDepartment(key, referral.Departments[i], stub)
		if err != nil {
			return []byte("Count not index the bytes by department from the value: " + value + " on the ledger"), err
		}
	}
	
	
	return nil, nil
}

func (t *ReferralChaincode) processCommaDelimitedReferrals(delimitedReferrals string, stub *shim.ChaincodeStub) ([]byte, error) {
	commaDelimitedReferrals := strings.Split(delimitedReferrals, ",")

	referralResultSet := "["
	
	for i := range commaDelimitedReferrals {
		valAsbytes, err := stub.GetState(commaDelimitedReferrals[i])
		
		if err != nil {
			return nil, err
		}
		
		if i == 0 {
			referralResultSet = referralResultSet + BytesToString(valAsbytes)
		} else {
			referralResultSet = referralResultSet + "," + BytesToString(valAsbytes)
		}
	}
	
	referralResultSet += "]"
	return []byte(referralResultSet), nil
}

func (t *ReferralChaincode) searchByDepartment(department string, stub *shim.ChaincodeStub) ([]byte, error) {
	valAsbytes, err := stub.GetState(department)
	
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + department + "\"}"
		return nil, errors.New(jsonResp)
	}
	
	valAsbytes, err = t.processCommaDelimitedReferrals(BytesToString(valAsbytes), stub)
	
	if(err != nil) {
		return nil, err
	}
	
	return valAsbytes, nil
}

func (t *ReferralChaincode) searchByStatus(status string, stub *shim.ChaincodeStub) ([]byte, error) {
	valAsbytes, err := stub.GetState(status)
	
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + status + "\"}"
		return nil, errors.New(jsonResp)
	}
	
	valAsbytes, err = t.processCommaDelimitedReferrals(BytesToString(valAsbytes), stub)
	
	if(err != nil) {
		return nil, err
	}
	
	return valAsbytes, nil
}


// read - query function to read key/value pair
func (t *ReferralChaincode) read(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var key, jsonResp string
	var err error
	
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the key to query")
	}

	key = args[0]
	valAsbytes, err := stub.GetState(key)
	
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
		return []byte(jsonResp), err
	}
	
	if valAsbytes == nil {
		return []byte("Did not find entry for key: " + key), nil
	}
	return valAsbytes, nil
}