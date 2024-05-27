package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"log"
	"time"

	ei "biehdc.reegg/eggpb"
	"google.golang.org/protobuf/proto"
)

// this one is special according to op:
// The permanent "first contract" is a special case where it should never expire
// and it should also not appear after 5000 Soul Eggs
var permacontract *ei.Contract = func() *ei.Contract {
	var (
		ident         = "first-contract"
		name          = "Your First Contract"
		desc          = "We heard you are open to contract work! Help fill this order from the local pharmacy!"
		egg_          = ei.Egg_MEDICAL
		lensecs       = 14400.0
		tokeninterval = 5.0
		exptime       = 100000000.0
		maxsouleggs   = 5000.0
	)
	var (
		rwtype1     = ei.RewardType_GOLD
		rwamount1   = 500.0
		rwamount1_2 = 192.0
		delamount1  = 100000.0
		rwtype2     = ei.RewardType_PIGGY_FILL
		rwamount2   = 10000.0
		delamount2  = 5000000000.0
	)
	firstcontract := ei.Contract{
		Identifier:      &ident,
		Name:            &name,
		Description:     &desc,
		LengthSeconds:   &lensecs,
		Egg:             &egg_,
		MinutesPerToken: &tokeninterval,
		ExpirationTime:  &exptime,
		MaxSoulEggs:     &maxsouleggs,
		GoalSets: []*ei.Contract_GoalSet{
			{
				Goals: []*ei.Contract_Goal{
					{
						RewardType:   &rwtype1,
						RewardAmount: &rwamount1,
						TargetAmount: &delamount1,
					},
					{
						RewardType:   &rwtype2,
						RewardAmount: &rwamount2,
						TargetAmount: &delamount2,
					},
				},
			},
			{
				Goals: []*ei.Contract_Goal{
					{
						RewardType:   &rwtype1,
						RewardAmount: &rwamount1_2,
						TargetAmount: &delamount1,
					},
					{
						RewardType:   &rwtype2,
						RewardAmount: &rwamount2,
						TargetAmount: &delamount2,
					},
				},
			},
		},
	}
	return &firstcontract
}()

// Just a dupe of the other one, for quick testing
var permacontractCoop *ei.Contract = func() *ei.Contract {
	var (
		ident         = "first-contract-coop"
		name          = "Your First Contract Coop"
		desc          = "We heard you are open to contract work! Help fill this order from the local pharmacy!"
		egg_          = ei.Egg_MEDICAL
		lensecs       = 14400.0
		tokeninterval = 5.0
		exptime       = 100000000.0
		maxsouleggs   = 5000.0
	)
	var (
		rwtype1     = ei.RewardType_GOLD
		rwamount1   = 500.0
		rwamount1_2 = 192.0
		delamount1  = 100000.0
		rwtype2     = ei.RewardType_PIGGY_FILL
		rwamount2   = 10000.0
		delamount2  = 5000000000.0
	)
	var (
		coopallowed = true
		maxcoop     = uint32(4)
	)
	firstcontract := ei.Contract{
		Identifier:      &ident,
		Name:            &name,
		Description:     &desc,
		LengthSeconds:   &lensecs,
		Egg:             &egg_,
		MinutesPerToken: &tokeninterval,
		ExpirationTime:  &exptime,
		MaxSoulEggs:     &maxsouleggs,
		CoopAllowed:     &coopallowed,
		MaxCoopSize:     &maxcoop,
		GoalSets: []*ei.Contract_GoalSet{
			{
				Goals: []*ei.Contract_Goal{
					{
						RewardType:   &rwtype1,
						RewardAmount: &rwamount1,
						TargetAmount: &delamount1,
					},
					{
						RewardType:   &rwtype2,
						RewardAmount: &rwamount2,
						TargetAmount: &delamount2,
					},
				},
			},
			{
				Goals: []*ei.Contract_Goal{
					{
						RewardType:   &rwtype1,
						RewardAmount: &rwamount1_2,
						TargetAmount: &delamount1,
					},
					{
						RewardType:   &rwtype2,
						RewardAmount: &rwamount2,
						TargetAmount: &delamount2,
					},
				},
			},
		},
	}
	return &firstcontract
}()

var (
	legacyRead  []*ei.Contract
	legacyWrite []*ei.Contract
	permanent   []*ei.Contract
)

func getContract(identifier string) *ei.Contract {
	if *permacontractCoop.Identifier == identifier {
		return permacontractCoop
	}
	for _, ct := range legacyRead {
		if *ct.Identifier == identifier {
			return ct
		}
	}
	return nil
}

//go:embed contracts.json
var contracts_json []byte

func generateContracts(workingpath string) {
	type upstream struct {
		Id    string `json:"id"`
		Proto string `json:"proto"`
	}

	// parse all the entries of the og file into an array
	var data []upstream
	err := json.Unmarshal(contracts_json, &data)
	if err != nil {
		log.Panic(err)
	}

	// decode the protobuf messages into proper structs
	for _, contract := range data {
		cnt, err := base64.StdEncoding.DecodeString(contract.Proto)
		if err != nil {
			log.Panic(err)
		}

		// we do it once for reading from
		var protoRead ei.Contract
		err = proto.Unmarshal(cnt, &protoRead)
		if err != nil {
			log.Panic(err)
		}
		legacyRead = append(legacyRead, &protoRead)

		// and we need this one to write to for dispatch
		var protodWrite ei.Contract
		err = proto.Unmarshal(cnt, &protodWrite)
		if err != nil {
			log.Panic(err)
		}
		legacyWrite = append(legacyWrite, &protodWrite)
	}

	permanent = []*ei.Contract{namechangecontract, permacontract, permacontractCoop}

	//log.Printf("Loaded %d \"Leggacy\" contracts, %d to-schedule contracts", len(legacyRead), len(normal))
	log.Printf("Loaded %d contracts", len(legacyRead))
}

func (egg *eggstore) updateContracts(t time.Time) []*ei.Contract {
	var activecontracts []*ei.Contract
	activecontracts = permanent
	cmon, cday := t.Month(), t.Day()

	const (
		q0 = 0
		q1 = 8
		q2 = 16
		q3 = 24
		q4 = 32
	)

	var lower, upper int
	switch {
	case cday >= q0 && cday < q1:
		lower = q0
		upper = q1
	case cday >= q1 && cday < q2:
		lower = q1
		upper = q2
	case cday >= q2 && cday < q3:
		lower = q2
		upper = q3
	case cday >= q3 && cday < q4:
		lower = q3
		upper = q4
	}

	// this is nether accurate to the original nor really correct
	// but it is good enough for myself and makes a constant feed
	// of new contracts for players
	for ii, ctread := range legacyRead {
		cttime := time.Unix(int64(*ctread.ExpirationTime), 0)
		if cttime.Month() != cmon {
			//log.Printf("ignoring %s, wrong month %d", *ctread.Identifier, cttime.Month())
			continue // wrong month
		}

		if cttime.Day() > lower && cttime.Day() < upper {
			ct := legacyWrite[ii]
			exp := float64(upper-cday) * 24 * 60 * 60
			ct.ExpirationTime = &exp
			activecontracts = append(activecontracts, ct)

			//log.Printf("have contract %s", *ct.Identifier)
			//log.Printf("expires at %f", exp)
		} else {
			//log.Printf("ignoring %s, wrong day %d", *ctread.Identifier, cttime.Day())
			continue // wrong day area
		}
	}

	return activecontracts
}

func (egg *eggstore) currentContracts(_ *ei.GetPeriodicalsRequest) *ei.ContractsResponse {
	var cr ei.ContractsResponse

	cr.WarningMessage = &egg.motd
	cr.Contracts = egg.updateContracts(time.Now())

	return &cr
}

func (egg *eggstore) path_get_contracts(decoded []byte) []byte {
	crreq := ei.ContractsRequest{}
	err := proto.Unmarshal(decoded, &crreq)
	if err != nil {
		log.Printf("cant unmarshal ContractsRequest: %s", err)
		return nil
	}

	// log.Printf("ContractsRequest: %s", crreq.String())

	crresp := egg.currentContracts(nil)

	resp, err := proto.Marshal(crresp)
	if err != nil {
		log.Printf("failed to marshal path_get_contracts: %s", err.Error())
		return nil
	}

	return resp
}

/*
maybe we can abuse this to make yes or no questions in the future
its very simple, make a contract with the question
then you start it
if you fail it due to timeout it means no
if you archieve the goal of 1 egg laid it means yes

// start of it
eihandler.go:153: log_action: user_id:"aaa"  action_name:"start_contract"  data:{key:"contract_identifier"  value:"sneed-feed-and-seed-5"}  app:{version_str:"1.12.13"}
// fail
eihandler.go:153: log_action: user_id:"aaa"  action_name:"expired_contract"  data:{key:"contract_identifier"  value:"sneed-feed-and-seed-5"}  app:{version_str:"1.12.13"}
// success
eihandler.go:153: log_action: user_id:"aaa"  action_name:"contract_reward_collected"  data:{key:"contract_identifier"  value:"sneed-feed-and-seed-5"}  data:{key:"at_amount"  value:"1.000000"}  app:{version_str:"1.12.13"}
eihandler.go:153: log_action: user_id:"aaa"  action_name:"finish_contract"  data:{key:"contract_identifier"  value:"sneed-feed-and-seed-5"}  app:{version_str:"1.12.13"}

// issue, those need to be forced upon the user
// backup->contracts->ContractIdsSeen     []string
// backup->contracts->Archive             []*LocalContract
// backup->contracts->Farms               []*Backup_Simulation
// we might have to add a regular data cleaner thing
//
var (

	ident   = "sneed-feed-and-seed-6"
	name    = "Test motd contract for pushing messages 25"
	desc    = "Please consume hopium! 35"
	lensecs = 30.0
	exptime = 36374400.0 //almost 421 days
	//eggt    = ei.Egg_UNKNOWN

)
var (

	goal       = ei.GoalType_EGGS_LAID
	rwtype2    = ei.RewardType_CASH
	rwamount2  = 1.0
	delamount2 = 1.0

)

	fakemotdcontract := ei.Contract{
		Identifier:     &ident,
		Name:           &name,
		Description:    &desc,
		LengthSeconds:  &lensecs,
		ExpirationTime: &exptime,
		//Egg:            &eggt,
		GoalSets: []*ei.Contract_GoalSet{
			{
				Goals: []*ei.Contract_Goal{
					{
						Type:         &goal,
						TargetAmount: &delamount2,
						RewardType:   &rwtype2,
						RewardAmount: &rwamount2,
					},
				},
			},
		},
	}
*/
