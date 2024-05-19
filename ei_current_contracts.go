package main

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
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

const _7days = float64(604800) // seconds
const _4days = float64(345600) // seconds
var (
	ALL_SOLO_CONTRACTS = true //fixme: TODO: Co-op contracts, toggle this off when we have them
	contractEpoch      = 1714867200
	legacy             []*ei.Contract
	normal             []*ei.Contract
	permanent          []*ei.Contract
)

func generateContracts(workingpath string) {
	joined := filepath.Join(workingpath, "contracts_go.json")
	contractsfile, err := os.Open(joined)
	if err != nil {
		log.Panic(err)
	}
	defer contractsfile.Close()
	bytes, err := io.ReadAll(contractsfile)
	if err != nil {
		log.Panic(err)
	}

	var contracts []*ei.Contract
	// lets hope this works
	err = json.Unmarshal(bytes, &contracts)
	if err != nil {
		log.Panic(err)
	}

	permanent = append(permanent, permacontract)

	const scaler = float64(1.0)
	tmpbool := false
	for i, ct := range contracts {
		// do some modifications
		if ALL_SOLO_CONTRACTS {
			ct.CoopAllowed = &tmpbool
			// op comment:
			// Not all contracts are made equal. If we divide it at an absolute, it becomes too easy
			// Still need to pinpoint the ratio based on experience.
			if ct.MaxCoopSize != nil {
				scalefactor := float64(*ct.MaxCoopSize) * 0.35
				if scalefactor > 1.0 {
					scalefactor = 1.0
				}
				for _, gs := range ct.GoalSets {
					for _, goal := range gs.Goals {
						*goal.TargetAmount = *goal.TargetAmount * scalefactor
					}
				}
			}
		}
		// add to places
		exp := float64((_7days * float64(i+1)) - float64(contractEpoch))
		ct.ExpirationTime = &exp
		legacy = append(legacy, ct)
	}

	log.Printf("Loaded %d \"Leggacy\" contracts, %d to-schedule contracts", len(legacy), len(normal))
}

func (egg *eggstore) updateContracts(t time.Time) []*ei.Contract {
	//fixme: TODO: Shift the epoch when a full run of leggacys is done.

	var activecontracts []*ei.Contract
	activecontracts = permanent

	timesinceepoch := t.Unix() - int64(contractEpoch)
	for ii, ct := range legacy {
		i := ii + 1

		var factor1 float64
		factor1 = _7days * math.Ceil(float64(i)/2.0)

		var factor2 float64
		if i%2 != 0 {
			factor2 = _4days
		}
		expirytime := factor1 + factor2 - float64(timesinceepoch)

		if expirytime < 0 {
			// its expired, get next one
			continue
		}

		if expirytime > _7days {
			// its next weeks contract, dont process more
			break
		}

		ct.ExpirationTime = &expirytime
		activecontracts = append(activecontracts, ct)
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
