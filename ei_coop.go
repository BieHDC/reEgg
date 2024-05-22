package main

import (
	"log"
	"slices"
	"time"

	ei "biehdc.reegg/eggpb"
	genericsync "biehdc.reegg/genericsyncmap"
	"biehdc.reegg/lockmap"
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
}

var members = lockmap.MakeLockMap[string, []usermemberinfo]()

func getMembersInGroup(coopname string) []usermemberinfo {
	var membersingroup []usermemberinfo

	members.LockedRange(func(k string, v []usermemberinfo) bool {
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

func countMembersInGroup(coopname string) int {
	return len(getMembersInGroup(coopname))
}

func getCoopMemberships(userid string) []string {
	var memberships []string

	userinfo, _ := members.LockedLoad(userid)
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

var coopgames = lockmap.MakeLockMap[string, contractGame]()

var coopgifts = lockmap.MakeLockMap[string, []*ei.ContractCoopStatusResponse_CoopGift]()

type coopStatusEx struct {
	lastvisit int64
	ccsur     *ei.ContractCoopStatusUpdateRequest
}

var coopstatus genericsync.Map[string, coopStatusEx]

func queryCoop(req *ei.QueryCoopRequest) *ei.QueryCoopResponse {
	var (
		exists          = false
		full            = false
		differentleague = false
		banned          = false
	)
	resp := ei.QueryCoopResponse{
		Exists:          &exists,
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

	lobby, exists := coopgames.LockedLoad(*req.CoopIdentifier)
	if !exists {
		//failed = "no lobby"
		return &resp
	}
	exists = true

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

	// check if lobby contract and requested contract match
	if lobby.ContractIdentifier != *req.ContractIdentifier {
		// assume bad actor
		banned = true
		//failed = "contract identifier mismatch"
		return &resp
	}

	// check if it is full
	if ct.MaxCoopSize != nil {
		if countMembersInGroup(*req.ContractIdentifier) >= int(*ct.MaxCoopSize) {
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

	now := time.Now().Unix()
	// calc how much time remeaning for it
	stamp := float64(now) - *contract.LengthSeconds + *req.SecondsRemaining

	// check if the group name is already used -> reject
	_, exists := coopgames.LockAndLoad(*req.CoopIdentifier)
	if exists {
		//failed = "-"
		message = "This name is already taken"
		return &resp
	}

	// create the coop group
	coopgames.StoreAndUnlock(*req.CoopIdentifier, contractGame{
		CoopIdentifier:     *req.CoopIdentifier,
		ContractIdentifier: *contract.Identifier,
		League:             *req.League,
		Stamp:              stamp,
		Owner:              *req.UserId,
		Public:             false,
	})

	// add the membership
	userinfo, _ := members.LockAndLoad(*req.UserId)
	members.StoreAndUnlock(*req.UserId, append(userinfo, usermemberinfo{
		Deviceid:    *req.UserId,
		CoopName:    *req.CoopIdentifier,
		DisplayName: *req.UserName,
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
		failed := "LOG"
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

	lobby, exists := coopgames.LockedLoad(*req.CoopIdentifier)
	if !exists {
		//failed = "group not exists"
		return &resp
	}

	if lobby.ContractIdentifier != *req.ContractIdentifier {
		//failed = fmt.Sprintf("contract identifier mismatch: expected %s got %s", lobby.ContractIdentifier, *req.ContractIdentifier)
		return &resp
	}

	resp.Public = &lobby.Public
	resp.CreatorId = &lobby.Owner
	stamp := time.Now().Unix()
	rem := lobby.Stamp + *contract.LengthSeconds - float64(stamp)
	resp.SecondsRemaining = &rem

	members := getMembersInGroup(*req.CoopIdentifier)
	for _, member := range members {
		contr := ei.ContractCoopStatusResponse_ContributionInfo{}
		contr.UserId = &member.Deviceid
		contr.UserName = &member.DisplayName

		status, exists := coopstatus.Load(member.Deviceid)
		if exists {
			active := false
			if !(stamp-status.lastvisit >= 86400) {
				active = true
			}
			contr.Active = &active
			contr.ContributionAmount = status.ccsur.Amount
			totalAmount += *status.ccsur.Amount
			contr.ContributionRate = status.ccsur.Rate
			contr.SoulPower = status.ccsur.SoulPower
			contr.BoostTokens = status.ccsur.BoostTokens
		}

		contributors = append(contributors, &contr)
	}
	resp.Contributors = contributors

	resp.Gifts, _ = coopgifts.LockedLoadAndDelete(*req.UserId)

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

	coopstatus.Store(*req.UserId, coopStatusEx{
		lastvisit: time.Now().Unix(),
		ccsur:     req,
	})
	finalised = true

	return &resp
}

func joinCoop(req *ei.JoinCoopRequest) *ei.JoinCoopResponse {
	var (
		success = false
		message = "Group not found"
	)
	resp := ei.JoinCoopResponse{
		Success: &success,
		Message: &message,
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
	lobby, exists := coopgames.LockedLoad(*req.CoopIdentifier)
	if !exists {
		//failed = fmt.Sprintf("coopIdentifier bad: %s", *req.CoopIdentifier)
		return &resp
	}

	return joinCoop2(req, lobby)
}

func joinCoop2(req *ei.JoinCoopRequest, lobby contractGame) *ei.JoinCoopResponse {
	var (
		success          = false
		message          = "contract identifier mismatch"
		banned           = false
		coopIdentifier   = lobby.CoopIdentifier
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

	if lobby.ContractIdentifier != *req.ContractIdentifier {
		//failed = "contract identifier mismatch"
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
	now := time.Now().Unix()

	// remaining seconds
	rem := lobby.Stamp + *contract.LengthSeconds - float64(now)
	resp.SecondsRemaining = &rem

	// add the membership
	userinfo, _ := members.LockAndLoad(*req.UserId)
	members.StoreAndUnlock(*req.UserId, append(userinfo, usermemberinfo{
		Deviceid:    *req.UserId,
		CoopName:    coopIdentifier,
		DisplayName: *req.UserName,
	}))

	// success
	success = true

	return &resp
}

func autoJoinCoop(req *ei.AutoJoinCoopRequest) *ei.JoinCoopResponse {
	var (
		success = false
		message = ""
	)
	resp := ei.JoinCoopResponse{
		Success: &success,
		Message: &message,
	}
	/*
		failed := "LOG"
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
	coopgames.LockedRange(func(_ string, v contractGame) bool {
		if v.Public {
			joinresp = joinCoop2(joinreq, v)
			if *joinresp.Success == true {
				return false // stop iterating
			}
		}
		return true // keep searching
	})

	if joinresp != nil && *joinresp.Success {
		return joinresp
	}

	message = "No Lobby found"
	return &resp
}

func leaveCoop(req *ei.LeaveCoopRequest) []byte {
	/*
		failed := "LOG"
		defer func() {
			if failed != "" {
				log.Printf("leaveCoop failed: %s", failed)
				log.Printf("leaveCoop Req: %s", req.String())
				log.Printf("leaveCoop Resp: None")
			}
		}()
	*/

	userinfo, _ := members.LockAndLoad(*req.PlayerIdentifier)
	slices.DeleteFunc(userinfo, func(ui usermemberinfo) bool {
		if ui.CoopName == *req.CoopIdentifier {
			return true
		}
		return false
	})
	members.StoreAndUnlock(*req.PlayerIdentifier, userinfo)

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

	lobby, _, unlocker := coopgames.LockLoadWithUnlockerFunc(*req.CoopIdentifier)
	defer unlocker()
	if lobby.Owner != *req.RequestingUserId {
		//failed = "attacker"
		message = "Only the creator can change the permissions"
		return &resp
	}

	lobby.Public = *req.Public
	coopgames.StoreWhenWithUnlocker(*req.CoopIdentifier, lobby)

	success = true
	message = "Success"

	return &resp
}

func giftPlayerCoop(req *ei.GiftPlayerCoopRequest) []byte {
	failed := ""
	/*
		defer func() {
			if failed != "" {
				log.Printf("giftPlayerCoop failed: %s", failed)
				log.Printf("giftPlayerCoop Req: %s", req.String())
				log.Printf("giftPlayerCoop Resp: None")
			}
		}()
	*/

	// fixme: # TODO: How do we validate the player even has as many boost tokens as they are about to gift?
	// RequestingUserId sender
	// PlayerIdentifier receiver

	// check if lobby exists
	lobby, exists := coopgames.LockedLoad(*req.CoopIdentifier)
	if !exists {
		failed = "Coop not fouund"
		return []byte(failed)
	}

	// check if coop and contract match
	if lobby.ContractIdentifier != *req.ContractIdentifier {
		failed = "contract identifier mismatch"
		return []byte(failed)
	}

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
		UserId:   req.RequestingUserId,
		UserName: req.RequestingUserName,
		Amount:   req.Amount,
	}

	currentgifts, _ := coopgifts.LockAndLoad(*req.PlayerIdentifier)
	currentgifts = append(currentgifts, &gift)
	coopgifts.StoreAndUnlock(*req.PlayerIdentifier, currentgifts)

	return []byte("Chuck") // it should expect nothing in response
}
