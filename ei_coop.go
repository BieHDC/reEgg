package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math/rand/v2"
	"slices"
	"strings"
	"time"
	"unicode"

	ei "biehdc.reegg/eggpb"
	"google.golang.org/protobuf/proto"
)

func (egg *eggstore) path_query_coop(decoded []byte) []byte {
	req := ei.QueryCoopRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal QueryCoopRequest: %s", err)
		return nil
	}

	resp, err := proto.Marshal(egg.queryCoop(&req))
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

	resp, err := proto.Marshal(egg.createCoop(&req))
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

	resp, err := proto.Marshal(egg.coopStatus(&req))
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

	resp, err := proto.Marshal(egg.updateCoopStatus(&req))
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

	resp, err := proto.Marshal(egg.joinCoop(&req))
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

	resp, err := proto.Marshal(egg.autoJoinCoop(&req))
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

	return egg.leaveCoop(&req)
}

func (egg *eggstore) path_update_coop_permissions(decoded []byte) []byte {
	req := ei.UpdateCoopPermissionsRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal UpdateCoopPermissionsRequest: %s", err)
		return nil
	}

	resp, err := proto.Marshal(egg.updateCoopPermissions(&req))
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

	return egg.giftPlayerCoop(&req)
}

func (egg *eggstore) path_kick_player_coop(decoded []byte) []byte {
	req := ei.KickPlayerCoopRequest{}
	err := proto.Unmarshal(decoded, &req)
	if err != nil {
		log.Printf("cant unmarshal KickPlayerCoopRequest: %s", err)
		return nil
	}

	return egg.kickPlayerCoop(&req)
}

type usermemberinfo struct {
	Deviceid    string
	CoopName    string
	DisplayName string
	PrivacyId   string
	Lastvisit   int64
	Ccsur       *ei.ContractCoopStatusUpdateRequest
}

func (egg *eggstore) getMembersInGroup(coopname string) []usermemberinfo {
	var membersingroup []usermemberinfo

	egg.members.LockedRange(func(k string, v []usermemberinfo) bool {
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

func (egg *eggstore) countMembersInGroup(coopname string) int {
	return len(egg.getMembersInGroup(coopname))
}

func (egg *eggstore) getCoopMemberships(userid string) []string {
	var memberships []string

	userinfo, _ := egg.members.LockedLoad(userid)
	for _, ui := range userinfo {
		memberships = append(memberships, ui.CoopName)
	}

	return memberships
}

func (egg *eggstore) isPlayerInCoop(userid string, groupname string) bool {
	memberships := egg.getCoopMemberships(userid)

	for _, group := range memberships {
		if group == groupname {
			return true
		}
	}

	return false
}

func calculatePrivacyId(deviceid string) string {
	sha := sha256.Sum256([]byte(deviceid))
	return fmt.Sprintf("%x", sha[8:16])
}

// also implicitly confirms the player is in the lobby
func (egg *eggstore) deviceIdFromPrivacyId(coopname, privacyid string) string {
	members := egg.getMembersInGroup(coopname)

	for _, member := range members {
		if member.PrivacyId == privacyid {
			return member.Deviceid
		}
	}

	return ""
}

type contractGame struct {
	CoopIdentifier     string
	ContractIdentifier string
	League             uint32
	Stamp              float64
	Owner              string
	Public             bool
}

func isValidDeviceId(s string) bool {
	if len(s) != 16 {
		return false
	}
	for _, r := range s {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
			return false
		}
	}
	return true
}

func isValidDisplayName(s string) bool {
	isSpecial := func(r rune) bool {
		for _, cc := range "-_[]() " {
			if r == cc {
				return true
			}
		}
		return false
	}
	ll := len(s)
	if ll < 1 || ll > 20 {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) {
			continue
		}
		if unicode.IsNumber(r) {
			continue
		}
		if isSpecial(r) {
			continue
		}
		// it was none of those
		return false
	}
	return true
}

func (egg *eggstore) queryCoop(req *ei.QueryCoopRequest) *ei.QueryCoopResponse {
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

	lobby, exists := egg.coopgames.LockedLoad(*req.CoopIdentifier)
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
	contract := getContract(*req.ContractIdentifier)
	if contract == nil {
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
	if contract.MaxCoopSize != nil {
		if egg.countMembersInGroup(*req.ContractIdentifier) >= int(*contract.MaxCoopSize) {
			full = true
		}
	}

	return &resp
}

func (egg *eggstore) createCoop(req *ei.CreateCoopRequest) *ei.CreateCoopResponse {
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

	if ncrr := egg.checkUsernameContract(req); ncrr != nil {
		// this was a username change request, leave
		// the return values will be handled by the function
		//failed = "username change request"
		return ncrr
	}

	if req.CoopIdentifier == nil {
		//failed = "no coop identifier"
		return &resp
	}

	if req.ContractIdentifier == nil {
		//failed = "-"
		message = "Bad Contract Identifier"
		return &resp
	}

	if !isValidDeviceId(*req.UserId) {
		//failed = "bad deviceid"
		message = "Bad DeviceID"
		return &resp
	}

	if !isValidDisplayName(*req.UserName) {
		//failed = "bad username"
		message = "Invalid Username"
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
	_, exists, unlocker := egg.coopgames.LockLoadWithUnlockerFunc(*req.CoopIdentifier)
	if exists {
		unlocker()
		//failed = "-"
		message = "This name is already taken"
		return &resp
	}

	// create the coop group
	egg.coopgames.StoreWhenWithUnlocker(*req.CoopIdentifier, contractGame{
		CoopIdentifier:     *req.CoopIdentifier,
		ContractIdentifier: *contract.Identifier,
		League:             *req.League,
		Stamp:              stamp,
		Owner:              *req.UserId,
		Public:             false,
	})
	unlocker()

	// add the membership
	userinfo, _ := egg.members.LockAndLoad(*req.UserId)
	egg.members.StoreAndUnlock(*req.UserId, append(userinfo, usermemberinfo{
		Deviceid:    *req.UserId,
		CoopName:    *req.CoopIdentifier,
		DisplayName: *req.UserName,
		PrivacyId:   calculatePrivacyId(*req.UserId),
	}))

	// great success
	successbool = true
	message = "Coop Lobby created"

	return &resp
}

func (egg *eggstore) coopStatus(req *ei.ContractCoopStatusRequest) *ei.ContractCoopStatusResponse {
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

	if !isValidDeviceId(*req.UserId) {
		//failed = "bad deviceid"
		return &resp
	}

	contract := getContract(*req.ContractIdentifier)
	if contract == nil {
		//failed = "no contract"
		return &resp
	}

	lobby, exists := egg.coopgames.LockedLoad(*req.CoopIdentifier)
	if !exists {
		//failed = "group not exists"
		return &resp
	}

	if lobby.ContractIdentifier != *req.ContractIdentifier {
		//failed = fmt.Sprintf("contract identifier mismatch: expected %s got %s", lobby.ContractIdentifier, *req.ContractIdentifier)
		return &resp
	}

	resp.Public = &lobby.Public
	if lobby.Owner == *req.UserId {
		// do not obfuscate ourselves to ourselves
		resp.CreatorId = &lobby.Owner
	} else {
		privacyid := calculatePrivacyId(lobby.Owner)
		resp.CreatorId = &privacyid
	}
	stamp := time.Now().Unix()
	rem := lobby.Stamp + *contract.LengthSeconds - float64(stamp)
	resp.SecondsRemaining = &rem

	var (
		amountAcum = 0.0
		rateAcum   = 0.0
		soulAcum   = 0.0
	)
	members := egg.getMembersInGroup(*req.CoopIdentifier)
	for _, member := range members {
		contr := ei.ContractCoopStatusResponse_ContributionInfo{}
		// do not obfuscate ourselves to ourselves
		if member.Deviceid == *req.UserId {
			contr.UserId = &member.Deviceid
		} else {
			contr.UserId = &member.PrivacyId
		}

		customname, exists := egg.usernames.Load(member.Deviceid)
		if exists {
			contr.UserName = &customname
		} else {
			contr.UserName = &member.DisplayName
		}

		active := false
		if !(stamp-member.Lastvisit >= 86400) {
			active = true
		}
		contr.Active = &active

		if member.Ccsur != nil {
			contr.ContributionAmount = member.Ccsur.Amount
			totalAmount += *member.Ccsur.Amount
			contr.ContributionRate = member.Ccsur.Rate
			contr.SoulPower = member.Ccsur.SoulPower
			contr.BoostTokens = member.Ccsur.BoostTokens
			//
			amountAcum += *member.Ccsur.Amount
			rateAcum += *member.Ccsur.Rate
			soulAcum += *member.Ccsur.SoulPower
			//log.Printf("%.0f %.0f %.0f", *member.Ccsur.Amount, *member.Ccsur.Rate, *member.Ccsur.SoulPower)
		}

		contributors = append(contributors, &contr)
	}
	numMembers := len(members)
	missing := int(*contract.MaxCoopSize) - numMembers
	if missing > 0 {
		amountAcum /= float64(numMembers)
		rateAcum /= float64(numMembers)
		soulAcum /= float64(numMembers)
		bots, amountContributed := makeBotPlayers(missing, amountAcum, rateAcum, soulAcum)
		totalAmount += amountContributed
		contributors = append(contributors, bots...)
	}

	resp.Contributors = contributors

	resp.Gifts, _ = egg.coopgifts.LockedLoadAndDelete(*req.UserId)

	return &resp
}

func randInBetween[T float64](min, max T) T {
	return T(rand.N(int(max-min)) + int(min))
}

func randJittered[T float64](initial T, jitter T) T {
	return randInBetween(initial-jitter, initial+jitter)
}

const botPrefix = "Botman"

// looks legit enough
func makeBotPlayers(num int, amountMed, rateMed, soulMed float64) ([]*ei.ContractCoopStatusResponse_ContributionInfo, float64) {
	var bots []*ei.ContractCoopStatusResponse_ContributionInfo
	var amountContributed float64

	for i := range num {
		var (
			userid       = fmt.Sprintf("%s %d", botPrefix, i)
			active       = true
			alwaysZero   = uint32(0)
			jitterAmount = randJittered(amountMed, 1500)
			jitterRate   = randJittered(rateMed, 50)
			jitterSoul   = randJittered(soulMed, 3)
		)
		bot := ei.ContractCoopStatusResponse_ContributionInfo{
			UserId:             &userid,
			UserName:           &userid,
			Active:             &active,
			ContributionAmount: &jitterAmount,
			ContributionRate:   &jitterRate,
			SoulPower:          &jitterSoul,
			BoostTokens:        &alwaysZero,
		}

		amountContributed += jitterAmount
		bots = append(bots, &bot)
	}

	return bots, amountContributed
}

func (egg *eggstore) updateCoopStatus(req *ei.ContractCoopStatusUpdateRequest) *ei.ContractCoopStatusUpdateResponse {
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

	if !isValidDeviceId(*req.UserId) {
		//failed = "bad deviceid"
		return &resp
	}

	userinfo, _ := egg.members.LockAndLoad(*req.UserId)
	for i, ui := range userinfo {
		if ui.CoopName == *req.CoopIdentifier {
			userinfo[i].Lastvisit = time.Now().Unix()
			userinfo[i].Ccsur = req
			break
		}
	}
	egg.members.StoreAndUnlock(*req.UserId, userinfo)

	finalised = true

	return &resp
}

func (egg *eggstore) joinCoop(req *ei.JoinCoopRequest) *ei.JoinCoopResponse {
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

	if !isValidDeviceId(*req.UserId) {
		//failed = "bad deviceid"
		message = "Bad DeviceID"
		return &resp
	}

	if !isValidDisplayName(*req.UserName) {
		//failed = "bad username"
		message = "Invalid Username"
		return &resp
	}

	if isUsernameChangeContract(*req.ContractIdentifier) {
		//failed = "this is a name change event, reject actual join"
		message = "You may now return 1" // does not show up
		return &resp
	}

	// check if coop group exists
	lobby, exists := egg.coopgames.LockedLoad(*req.CoopIdentifier)
	if !exists {
		//failed = fmt.Sprintf("coopIdentifier bad: %s", *req.CoopIdentifier)
		return &resp
	}

	return egg.joinCoop2(req, lobby)
}

func (egg *eggstore) joinCoop2(req *ei.JoinCoopRequest, lobby contractGame) *ei.JoinCoopResponse {
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
	num := egg.countMembersInGroup(coopIdentifier)
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
	userinfo, _ := egg.members.LockAndLoad(*req.UserId)
	egg.members.StoreAndUnlock(*req.UserId, append(userinfo, usermemberinfo{
		Deviceid:    *req.UserId,
		CoopName:    coopIdentifier,
		DisplayName: *req.UserName,
		PrivacyId:   calculatePrivacyId(*req.UserId),
	}))

	// success
	success = true

	return &resp
}

func (egg *eggstore) autoJoinCoop(req *ei.AutoJoinCoopRequest) *ei.JoinCoopResponse {
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

	if !isValidDeviceId(*req.UserId) {
		//failed = "bad deviceid"
		message = "Bad DeviceID"
		return &resp
	}

	if !isValidDisplayName(*req.UserName) {
		//failed = "bad username"
		message = "Invalid Username"
		return &resp
	}

	if isUsernameChangeContract(*req.ContractIdentifier) {
		//failed = "this is a name change event, reject actual join"
		message = "You may now return 2" // does not show up
		return &resp
	}

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
	egg.coopgames.LockedRange(func(_ string, v contractGame) bool {
		if v.Public {
			joinresp = egg.joinCoop2(joinreq, v)
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

func (egg *eggstore) leaveCoop(req *ei.LeaveCoopRequest) []byte {
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

	if !isValidDeviceId(*req.PlayerIdentifier) {
		//failed = "bad deviceid"
		return []byte("bad deviceid")
	}

	userinfo, _ := egg.members.LockAndLoad(*req.PlayerIdentifier)
	userinfo = slices.DeleteFunc(userinfo, func(ui usermemberinfo) bool {
		if ui.CoopName == *req.CoopIdentifier {
			return true
		}
		return false
	})
	if len(userinfo) < 1 {
		// user is in no lobby anymore
		egg.members.DeleteAndUnlock(*req.PlayerIdentifier)
	} else {
		egg.members.StoreAndUnlock(*req.PlayerIdentifier, userinfo)
	}

	if egg.countMembersInGroup(*req.CoopIdentifier) < 1 {
		//log.Printf("lobby %s is empty, deleting...", *req.CoopIdentifier)
		egg.coopgames.LockedDelete(*req.CoopIdentifier)
	}

	return []byte("Sneed") // it should expect nothing in response
}

func (egg *eggstore) updateCoopPermissions(req *ei.UpdateCoopPermissionsRequest) *ei.UpdateCoopPermissionsResponse {
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

	if !isValidDeviceId(*req.RequestingUserId) {
		//failed = "bad deviceid"
		message = "Bad DeviceID"
		return &resp
	}

	lobby, _, unlocker := egg.coopgames.LockLoadWithUnlockerFunc(*req.CoopIdentifier)
	defer unlocker()
	if lobby.Owner != *req.RequestingUserId {
		//failed = "attacker"
		message = "Only the creator can change the permissions"
		return &resp
	}

	lobby.Public = *req.Public
	egg.coopgames.StoreWhenWithUnlocker(*req.CoopIdentifier, lobby)

	success = true
	message = "Success"

	return &resp
}

func (egg *eggstore) giftPlayerCoop(req *ei.GiftPlayerCoopRequest) []byte {
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

	// RequestingUserId sender
	// PlayerIdentifier receiver

	if !isValidDeviceId(*req.RequestingUserId) {
		//failed = "bad deviceid sender"
		return []byte("Bad Sender DeviceID")
	}

	if !isValidDeviceId(*req.PlayerIdentifier) {
		if strings.HasPrefix(*req.PlayerIdentifier, botPrefix) {
			//log.Printf("player %s wanted to gift a bot, return the gift", *req.RequestingUserId)
			gift := ei.ContractCoopStatusResponse_CoopGift{
				UserId:   req.PlayerIdentifier,
				UserName: req.PlayerIdentifier,
				Amount:   req.Amount,
			}
			currentgifts, _ := egg.coopgifts.LockAndLoad(*req.RequestingUserId)
			currentgifts = append(currentgifts, &gift)
			egg.coopgifts.StoreAndUnlock(*req.RequestingUserId, currentgifts)
		}
		//failed = "bad deviceid receiver"
		return []byte("Bad Receiver DeviceID")
	}

	if !isValidDisplayName(*req.RequestingUserName) {
		//failed = "bad username"
		return []byte("Invalid Sender Username")
	}

	// check if lobby exists
	lobby, exists := egg.coopgames.LockedLoad(*req.CoopIdentifier)
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
	if !egg.isPlayerInCoop(*req.RequestingUserId, *req.CoopIdentifier) {
		failed = "Sender not in Group"
		return []byte(failed)
	}

	// deobfuscate deviceid
	realdeviceid := egg.deviceIdFromPrivacyId(*req.CoopIdentifier, *req.PlayerIdentifier)
	if realdeviceid == "" {
		failed = "Receiver not in Group"
		return []byte(failed)
	}

	// check if receiver is in the same coop (confirmed by deviceIdFromPrivacyId already)
	//if !isPlayerInCoop(*req.PlayerIdentifier, *req.CoopIdentifier) {
	//	failed = "Receiver not in Group"
	//	return []byte(failed)
	//}

	senderprivacyid := calculatePrivacyId(*req.RequestingUserId)
	// insert gift into giftmap
	gift := ei.ContractCoopStatusResponse_CoopGift{
		UserId:   &senderprivacyid,
		UserName: req.RequestingUserName,
		Amount:   req.Amount,
	}
	currentgifts, _ := egg.coopgifts.LockAndLoad(realdeviceid)
	currentgifts = append(currentgifts, &gift)
	egg.coopgifts.StoreAndUnlock(realdeviceid, currentgifts)

	return []byte("Chuck") // it should expect nothing in response
}

func (egg *eggstore) kickPlayerCoop(req *ei.KickPlayerCoopRequest) []byte {
	failed := ""
	/*
		defer func() {
			if failed != "" {
				log.Printf("kickPlayerCoop failed: %s", failed)
				log.Printf("kickPlayerCoop Req: %s", req.String())
				log.Printf("kickPlayerCoop Resp: None")
			}
		}()
	*/

	// RequestingUserId sender
	// PlayerIdentifier receiver

	if !isValidDeviceId(*req.PlayerIdentifier) {
		//failed = "bad deviceid receiver"
		return []byte("Bad Receiver DeviceID")
	}

	if !isValidDeviceId(*req.RequestingUserId) {
		//failed = "bad deviceid sender"
		return []byte("Bad Sender DeviceID")
	}

	// check if lobby exists
	lobby, exists := egg.coopgames.LockedLoad(*req.CoopIdentifier)
	if !exists {
		failed = "Coop not fouund"
		return []byte(failed)
	}

	// check if sender is owner of the lobby
	if lobby.Owner != *req.RequestingUserId {
		failed = "only the owner is allowed to kick"
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
	if !egg.isPlayerInCoop(*req.RequestingUserId, *req.CoopIdentifier) {
		failed = "Sender not in Group"
		return []byte(failed)
	}

	// deobfuscate deviceid
	realdeviceid := egg.deviceIdFromPrivacyId(*req.CoopIdentifier, *req.PlayerIdentifier)
	if realdeviceid == "" {
		failed = "Receiver not in Group"
		return []byte(failed)
	}

	// check if receiver is in the same coop (confirmed by deviceIdFromPrivacyId already)
	//if !isPlayerInCoop(*req.PlayerIdentifier, *req.CoopIdentifier) {
	//	failed = "Receiver not in Group"
	//	return []byte(failed)
	//}

	// make the kicked player leave
	return egg.leaveCoop(&ei.LeaveCoopRequest{
		ContractIdentifier: req.ContractIdentifier,
		CoopIdentifier:     req.CoopIdentifier,
		PlayerIdentifier:   &realdeviceid,
		ClientVersion:      req.ClientVersion,
	})
}
