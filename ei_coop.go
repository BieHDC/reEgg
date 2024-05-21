package main

import (
	"log"
	"slices"
	"time"

	ei "biehdc.reegg/eggpb"
	genericsync "biehdc.reegg/genericsyncmap"
	"google.golang.org/protobuf/proto"
)

func (egg *eggstore) path_query_coop(decoded []byte) []byte {
	req := ei.QueryCoopRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal QueryCoopRequest: %s", err)
		return nil
	}

	resp, err := proto.Marshal(queryCoop(&req))
	if err != nil {
		log.Printf("failed to marshal QueryCoopResponse: %s", err.Error())
		return nil
	}
	return resp
}

func (egg *eggstore) path_create_coop(decoded []byte) []byte {
	req := ei.CreateCoopRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal CreateCoopRequest: %s", err)
		return nil
	}

	resp, err := proto.Marshal(createCoop(&req))
	if err != nil {
		log.Printf("failed to marshal CreateCoopResponse: %s", err.Error())
		return nil
	}
	return resp
}

func (egg *eggstore) path_coop_status(decoded []byte) []byte {
	req := ei.ContractCoopStatusRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal ContractCoopStatusRequest: %s", err)
		return nil
	}

	resp, err := proto.Marshal(coopStatus(&req))
	if err != nil {
		log.Printf("failed to marshal ContractCoopStatusResponse: %s", err.Error())
		return nil
	}
	return resp
}

func (egg *eggstore) path_update_coop_status(decoded []byte) []byte {
	req := ei.ContractCoopStatusUpdateRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal ContractCoopStatusUpdateRequest: %s", err)
		return nil
	}

	resp, err := proto.Marshal(updateCoopStatus(&req))
	if err != nil {
		log.Printf("failed to marshal ContractCoopStatusUpdateResponse: %s", err.Error())
		return nil
	}
	return resp
}

func (egg *eggstore) path_join_coop(decoded []byte) []byte {
	req := ei.JoinCoopRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal JoinCoopRequest: %s", err)
		return nil
	}

	resp, err := proto.Marshal(joinCoop(&req))
	if err != nil {
		log.Printf("failed to marshal JoinCoopResponse: %s", err.Error())
		return nil
	}
	return resp
}

func (egg *eggstore) path_auto_join_coop(decoded []byte) []byte {
	req := ei.AutoJoinCoopRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal AutoJoinCoopRequest: %s", err)
		return nil
	}

	resp, err := proto.Marshal(autoJoinCoop(&req))
	if err != nil {
		log.Printf("failed to marshal JoinCoopResponse: %s", err.Error())
		return nil
	}
	return resp
}

func (egg *eggstore) path_leave_coop(decoded []byte) []byte {
	req := ei.LeaveCoopRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal LeaveCoopRequest: %s", err)
		return nil
	}

	return leaveCoop(&req)
}

func (egg *eggstore) path_update_coop_permissions(decoded []byte) []byte {
	req := ei.UpdateCoopPermissionsRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal UpdateCoopPermissionsRequest: %s", err)
		return nil
	}

	resp, err := proto.Marshal(updateCoopPermissions(&req))
	if err != nil {
		log.Printf("failed to marshal UpdateCoopPermissionsResponse: %s", err.Error())
		return nil
	}
	return resp
}

func (egg *eggstore) path_gift_player_coop(decoded []byte) []byte {
	req := ei.GiftPlayerCoopRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal GiftPlayerCoopRequest: %s", err)
		return nil
	}

	return giftPlayerCoop(&req)
}

type usermemberinfo struct {
	Deviceid    string
	CoopName    string
	DisplayName string
	LastVisit   time.Time
}

var members genericsync.Map[string, []usermemberinfo]

func countMembersInGroup(coopname string) int {
	var count int

	members.Range(func(_ string, v []usermemberinfo) bool {
		for _, mi := range v {
			if mi.CoopName == coopname {
				count++
				break // we can only be in it once
			}
		}
		return true
	})

	return count
}

var coopstatus genericsync.Map[string, *ei.ContractCoopStatusUpdateRequest]

func getMembersInGroup(coopname string) []usermemberinfo {
	var membersingroup []usermemberinfo

	members.Range(func(k string, v []usermemberinfo) bool {
		for _, mi := range v {
			if mi.CoopName == coopname {
				membersingroup = append(membersingroup, mi)
				break // we can only be in it once
			}
		}
		return true
	})

	return membersingroup
}

func getCoopMemberships(userid string) []string {
	var memberships []string

	userinfo, _ := members.Load(userid)
	for _, ui := range userinfo {
		memberships = append(memberships, ui.CoopName)
	}

	return memberships
}

func isPlayerInCoop(userid string, groupname string) bool {
	memberships := getCoopMemberships(userid)

	for _, group := range memberships {
		if group == groupname {
			return true
		}
	}

	return false
}

type contractGame struct {
	CoopIdentifier     string
	ContractIdentifier string
	League             uint32
	Stamp              float64
	Owner              string
	Public             bool
}

var coopgames genericsync.Map[string, *contractGame]

var coopgifts genericsync.Map[string, []*ei.ContractCoopStatusResponse_CoopGift]

func queryCoop(req *ei.QueryCoopRequest) *ei.QueryCoopResponse {
	var (
		groupexists     = false
		full            = false
		differentleague = false
		banned          = false
	)
	resp := ei.QueryCoopResponse{
		Exists:          &groupexists,
		Full:            &full,
		DifferentLeague: &differentleague,
		Banned:          &banned,
	}
	/*
		failed := ""
		defer func() {
			if failed != "" {
				log.Printf("queryCoop failed: %s", failed)
				log.Printf("queryCoop Req: %s", req.String())
				log.Printf("queryCoop Resp: %s", resp.String())
			}
		}()
	*/

	if req.ContractIdentifier == nil {
		//failed = "-"
		return &resp
	}

	lobby, exists := coopgames.Load(*req.CoopIdentifier)
	if !exists {
		//failed = "no lobby"
		return &resp
	}
	groupexists = true

	// check if its in a different league -> reject
	if lobby.League != *req.League {
		differentleague = true
		return &resp
	}

	// check if contract exists
	ct := getContract(*req.ContractIdentifier)
	if ct == nil {
		// idk what we would do for a bad contract, so say the user is banned
		// there isnt really a good way to deal with this
		banned = true
		//failed = "no contract"
		return &resp
	}

	// check if it is full
	num := countMembersInGroup(*req.ContractIdentifier)
	if ct.MaxCoopSize != nil {
		if num >= int(*ct.MaxCoopSize) {
			full = true
		}
	}

	return &resp
}

func createCoop(req *ei.CreateCoopRequest) *ei.CreateCoopResponse {
	var (
		successbool = false
		message     = "Bad Coop Identifier"
	)
	resp := ei.CreateCoopResponse{
		Success: &successbool,
		Message: &message,
	}
	/*
		failed := ""
		defer func() {
			if failed != "" {
				log.Printf("createCoop failed: %s", failed)
				log.Printf("createCoop Req: %s", req.String())
				log.Printf("createCoop Resp: %s", resp.String())
			}
		}()
	*/

	if req.CoopIdentifier == nil {
		//failed = "no coop identifier"
		return &resp
	}

	// check if the group name is already used -> reject
	_, exists := coopgames.Load(*req.CoopIdentifier)
	if exists {
		//failed = "-"
		message = "This name is already taken"
		return &resp
	}

	if req.ContractIdentifier == nil {
		//failed = "-"
		message = "Bad Contract Identifier"
		return &resp
	}

	// check if contract exists -> reject if not
	contract := getContract(*req.ContractIdentifier)
	if contract == nil {
		//failed = "-"
		message = "Requested Contract was not found"
		return &resp
	}

	now := time.Now()
	// calc how much time remeaning for it
	stamp := float64(now.Unix()) - *contract.LengthSeconds + *req.SecondsRemaining

	// create the coop group
	coopgames.Store(*req.CoopIdentifier, &contractGame{
		CoopIdentifier:     *req.CoopIdentifier,
		ContractIdentifier: *contract.Identifier, //fixme why not just a ptr to contract?
		League:             *req.League,
		Stamp:              stamp,
		Owner:              *req.UserId,
		Public:             false,
	})

	// add the membership
	userinfo, _ := members.Load(*req.UserId)
	members.Store(*req.UserId, append(userinfo, usermemberinfo{
		Deviceid:    *req.UserId,
		CoopName:    *req.CoopIdentifier,
		DisplayName: *req.UserName,
		LastVisit:   now,
	}))

	// great success
	successbool = true
	message = "Coop Lobby created"

	return &resp
}

func coopStatus(req *ei.ContractCoopStatusRequest) *ei.ContractCoopStatusResponse {
	var (
		totalAmount                 = 0.0
		contributors                []*ei.ContractCoopStatusResponse_ContributionInfo
		autoGenerated               = false
		allMembersReporting         = false
		gracePeriodSecondsRemaining = 888888.0
	)
	resp := ei.ContractCoopStatusResponse{
		ContractIdentifier:          req.ContractIdentifier,
		TotalAmount:                 &totalAmount,
		CoopIdentifier:              req.CoopIdentifier,
		Contributors:                contributors,
		AutoGenerated:               &autoGenerated,
		AllMembersReporting:         &allMembersReporting,
		GracePeriodSecondsRemaining: &gracePeriodSecondsRemaining,
	}
	/*
		failed := ""
		defer func() {
			if failed != "" {
				log.Printf("coopStatus failed: %s", failed)
				log.Printf("coopStatus Req: %s", req.String())
				log.Printf("coopStatus Resp: %s", resp.String())
			}
		}()
	*/

	contract := getContract(*req.ContractIdentifier)
	if contract == nil {
		//failed = "no contract"
		return &resp
	}

	group, _ := coopgames.Load(*req.CoopIdentifier)
	if group == nil {
		//failed = "group not exists"
		return &resp
	}

	resp.Public = &group.Public
	resp.CreatorId = &group.Owner
	rem := group.Stamp + *contract.LengthSeconds - float64(time.Now().Unix())
	resp.SecondsRemaining = &rem

	members := getMembersInGroup(*req.CoopIdentifier)
	for _, member := range members {
		var (
			active = true
		)
		contr := ei.ContractCoopStatusResponse_ContributionInfo{}
		contr.UserId = &member.Deviceid
		contr.UserName = &member.DisplayName
		contr.Active = &active

		status, _ := coopstatus.Load(member.Deviceid)
		if status != nil {
			contr.ContributionAmount = status.Amount
			totalAmount += *status.Amount
			contr.ContributionRate = status.Rate
			contr.SoulPower = status.SoulPower
			contr.BoostTokens = status.BoostTokens
		}

		contributors = append(contributors, &contr)
	}
	// not a pointer, need to reassign
	resp.Contributors = contributors

	resp.Gifts, _ = coopgifts.LoadAndDelete(*req.UserId)

	return &resp
}

func updateCoopStatus(req *ei.ContractCoopStatusUpdateRequest) *ei.ContractCoopStatusUpdateResponse {
	var (
		finalised = false
	)
	resp := ei.ContractCoopStatusUpdateResponse{
		Finalized: &finalised,
	}
	/*
		failed := "LOG"
		defer func() {
			if failed != "" {
				log.Printf("updateCoopStatus failed: %s", failed)
				log.Printf("updateCoopStatus Req: %s", req.String())
				log.Printf("updateCoopStatus Resp: %s", resp.String())
			}
		}()
	*/

	coopstatus.Store(*req.UserId, req)
	finalised = true

	return &resp
}

func joinCoop(req *ei.JoinCoopRequest) *ei.JoinCoopResponse {
	var (
		success          = false
		message          = "Group not found"
		banned           = false
		coopIdentifier   = "unknown"
		secondsRemaining = 5.0
	)
	resp := ei.JoinCoopResponse{
		Success:          &success,
		Message:          &message,
		Banned:           &banned,
		CoopIdentifier:   &coopIdentifier,
		SecondsRemaining: &secondsRemaining,
	}
	/*
		failed := ""
		defer func() {
			if failed != "" {
				log.Printf("joinCoop failed: %s", failed)
				log.Printf("joinCoop Req: %s", req.String())
				log.Printf("joinCoop Resp: %s", resp.String())
			}
		}()
	*/

	// check if coop group exists
	coopIdentifier = *req.CoopIdentifier
	lobby, exists := coopgames.Load(*req.CoopIdentifier)
	if !exists {
		//failed = fmt.Sprintf("coopIdentifier bad: %s", coopIdentifier)
		return &resp
	}

	// check if in the same league
	if *req.League != lobby.League {
		message = "Wrong League"
		return &resp
	}

	contract := getContract(*req.ContractIdentifier)
	num := countMembersInGroup(coopIdentifier)
	// check if full
	if contract != nil && contract.MaxCoopSize != nil && num >= int(*contract.MaxCoopSize) {
		//failed = "lobby full"
		message = "Lobby is Full"
		return &resp
	}

	// TODO bans
	// success, banned, remaining sec
	success = true

	group, _ := coopgames.Load(*req.CoopIdentifier)
	rem := group.Stamp + *contract.LengthSeconds - float64(time.Now().Unix())
	resp.SecondsRemaining = &rem

	// add the membership
	now := time.Now()
	userinfo, _ := members.Load(*req.UserId)
	members.Store(*req.UserId, append(userinfo, usermemberinfo{
		Deviceid:    *req.UserId,
		CoopName:    *req.CoopIdentifier,
		DisplayName: *req.UserName,
		LastVisit:   now,
	}))

	return &resp
}

func autoJoinCoop(req *ei.AutoJoinCoopRequest) *ei.JoinCoopResponse {
	var (
		success = false
		message = "Invalid Contract"
	)
	resp := ei.JoinCoopResponse{
		Success: &success,
		Message: &message,
	}
	/*
		failed := ""
		defer func() {
			if failed != "" {
				log.Printf("autoJoinCoop failed: %s", failed)
				log.Printf("autoJoinCoop Req: %s", req.String())
				log.Printf("autoJoinCoop Resp: %s", resp.String())
			}
		}()
	*/

	// convert the request
	joinreq := &ei.JoinCoopRequest{
		ContractIdentifier: req.ContractIdentifier,
		//CoopIdentifier: // handled special
		UserId:        req.UserId,
		UserName:      req.UserName,
		SoulPower:     req.SoulPower,
		League:        req.League,
		Platform:      req.Platform,
		ClientVersion: req.ClientVersion,
	}

	var joinresp *ei.JoinCoopResponse
	coopgames.Range(func(_ string, v *contractGame) bool {
		if v.Public {
			joinreq.CoopIdentifier = &v.CoopIdentifier
			joinresp = joinCoop(joinreq)
			if *joinresp.Success == true {
				return false // stop iterating
			}
		}
		return true // keep searching
	})

	if joinresp != nil && *joinresp.Success {
		return joinresp
	}

	// fixme if none found, create one
	// you just have to call createCoop
	message = "No Lobby found"
	return &resp
}

func leaveCoop(req *ei.LeaveCoopRequest) []byte {
	/*
		failed := "-"
		defer func() {
			if failed != "" {
				log.Printf("leaveCoop failed: %s", failed)
				log.Printf("leaveCoop Req: %s", req.String())
				log.Printf("leaveCoop Resp: None")
			}
		}()
	*/
	userinfo, _ := members.Load(*req.PlayerIdentifier)
	slices.DeleteFunc(userinfo, func(ui usermemberinfo) bool {
		if ui.CoopName == *req.CoopIdentifier {
			return true
		}
		return true
	})
	members.Store(*req.PlayerIdentifier, userinfo)

	return []byte("Sneed") // it should expect nothing in response
}

func updateCoopPermissions(req *ei.UpdateCoopPermissionsRequest) *ei.UpdateCoopPermissionsResponse {
	var (
		success = false
		message = "error"
	)
	resp := ei.UpdateCoopPermissionsResponse{
		Success: &success,
		Message: &message,
	}
	/*
		failed := ""
		defer func() {
			if failed != "" {
				log.Printf("updateCoopPermissions failed: %s", failed)
				log.Printf("updateCoopPermissions Req: %s", req.String())
				log.Printf("updateCoopPermissions Resp: %s", resp.String())
			}
		}()
	*/

	group, _ := coopgames.Load(*req.CoopIdentifier)
	if group.Owner != *req.RequestingUserId {
		//failed = "attacker"
		message = "Only the creator can change the permissions"
		return &resp
	}

	group.Public = *req.Public
	coopgames.Store(*req.CoopIdentifier, group)

	success = true
	message = "Success"

	return &resp
}

// fixme: not yet tested
func giftPlayerCoop(req *ei.GiftPlayerCoopRequest) []byte {
	failed := ""
	defer func() {
		if failed != "" {
			log.Printf("giftPlayerCoop failed: %s", failed)
			log.Printf("giftPlayerCoop Req: %s", req.String())
			log.Printf("giftPlayerCoop Resp: None")
		}
	}()

	// fixme: # TODO: How do we validate the player even has as many boost tokens as they are about to gift?
	// RequestingUserId sender
	// PlayerIdentifier receiver

	// check if group exists
	group, _ := coopgames.Load(*req.CoopIdentifier)
	if group == nil {
		failed = "Coop not fouund"
		return []byte(failed)
	}

	// check if coop and contract match?
	contract := getContract(*req.ContractIdentifier)
	if contract == nil {
		failed = "Contract not found"
		return []byte(failed)
	}

	// check if sender is in a coop
	if !isPlayerInCoop(*req.RequestingUserId, *req.CoopIdentifier) {
		failed = "Sender not in Group"
		return []byte(failed)
	}

	// check if receiver is in the same coop
	if !isPlayerInCoop(*req.PlayerIdentifier, *req.CoopIdentifier) {
		failed = "Receiver not in Group"
		return []byte(failed)
	}

	// insert gift into giftmap
	gift := ei.ContractCoopStatusResponse_CoopGift{
		UserId:   &failed,
		UserName: &failed,
		Amount:   nil,
	}

	currentgifts, _ := coopgifts.Load(*req.PlayerIdentifier)
	currentgifts = append(currentgifts, &gift)
	coopgifts.Store(*req.PlayerIdentifier, currentgifts)

	return []byte("Chuck") // it should expect nothing in response
}
