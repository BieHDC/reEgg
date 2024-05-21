package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	ei "biehdc.reegg/eggpb"
	"google.golang.org/protobuf/proto"
)

func (egg *eggstore) path_get_periodicals(decoded []byte) []byte {
	getperiodicals := ei.GetPeriodicalsRequest{}
	err := proto.Unmarshal(decoded, &getperiodicals)
	if err != nil {
		log.Printf("cant unmarshal GetPeriodicalsRequest: %s", err)
		return nil
	}

	if getperiodicals.UserId != nil {
		log.Printf("get_periodicals: %s", *getperiodicals.UserId)
	}

	perresp := ei.PeriodicalsResponse{
		Contracts: egg.currentContracts(&getperiodicals),
		Events:    egg.currentEvents(&getperiodicals),
		Gifts:     egg.serverGifts(&getperiodicals),
	}

	resp, err := proto.Marshal(&perresp)
	if err != nil {
		log.Printf("failed to marshal get_periodicals: %s", err.Error())
		return nil
	}

	return resp
}

func (egg *eggstore) handlepath_ei(w http.ResponseWriter, req *http.Request) error {
	var err error
	action := req.PathValue("subpath")
	req.ParseForm()

	var decoded []byte
	havedata := len(req.Form["data"])
	if havedata > 1 {
		log.Panicf("subpath is %q: we have a package with multiple data entries which we do not handle at this point", action)
	}
	if havedata > 0 {
		// strings.ReplaceAll is due to android bug wrong padding according to op
		decoded, err = base64.StdEncoding.DecodeString(strings.ReplaceAll(req.Form["data"][0], " ", "+"))
		if err != nil {
			log.Printf("cant base64 decode: %s", err)
			return errors.New("do not send me broken base64 strings")
		}
	}

	var resp []byte

	switch action {
	// userdata stuff
	case "first_contact":
		resp = egg.path_first_contact(decoded)
	case "save_backup":
		resp = egg.path_save_backup(decoded)
	case "user_data_info":
		resp = egg.path_user_data_info(decoded)

	// gameplay stuff
	case "get_periodicals":
		resp = egg.path_get_periodicals(decoded)
	case "daily_gift_info":
		resp = path_daily_gift_info(decoded)
	case "get_contracts":
		resp = egg.path_get_contracts(decoded)

	// coop stuff
	case "query_coop":
		resp = egg.path_query_coop(decoded)
	case "create_coop":
		resp = egg.path_create_coop(decoded)
	case "coop_status":
		resp = egg.path_coop_status(decoded)
	case "update_coop_status":
		resp = egg.path_update_coop_status(decoded)
	case "join_coop":
		resp = egg.path_join_coop(decoded)
	case "auto_join_coop":
		resp = egg.path_auto_join_coop(decoded)
	case "leave_coop":
		resp = egg.path_leave_coop(decoded)
	case "update_coop_permissions":
		resp = egg.path_update_coop_permissions(decoded)

	//misc stuff
	case "get_ad_config":
		//noop and ignored
		return errors.New("we dont advertise anymore")

	default:
		// something we arent dealing with yet
		return handleUnhandled("POST /ei/", action, req)
	}

	if resp != nil {
		buf := make([]byte, base64.StdEncoding.EncodedLen(len(resp)))
		base64.StdEncoding.Encode(buf, resp)
		//log.Printf("replying: %q", string(buf))
		w.Write(buf)
	}

	return nil
}

func (egg *eggstore) handlepath_eidata(w http.ResponseWriter, req *http.Request) error {
	var err error
	action := req.PathValue("subpath")
	req.ParseForm()

	var decoded []byte
	havedata := len(req.Form["data"])
	if havedata > 1 {
		log.Panicf("subpath is %q: we have a package with multiple data entries which we do not handle at this point", action)
	}
	if havedata > 0 {
		// strings.ReplaceAll is due to android bug wrong padding according to op
		decoded, err = base64.StdEncoding.DecodeString(strings.ReplaceAll(req.Form["data"][0], " ", "+"))
		if err != nil {
			log.Printf("cant base64 decode: %s", err)
			return errors.New("do not send me broken base64 strings")
		}
	}

	switch action {
	case "log_action":
		gena := ei.GenericAction{}
		err := proto.Unmarshal(decoded, &gena)
		if err != nil {
			log.Printf("cant unmarshal GenericAction: %s", err)
			return nil
		}
		log.Printf("log_action: %s", gena.String())

	default:
		return handleUnhandled("POST /ei_data/", action, req)
	}

	return nil
}

func handleUnhandled(path, action string, req *http.Request) error {
	log.Printf("subpath is %q", action)
	log.Println("unhandled action:", action)

	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Printf("failed to dump request: %s", err)
		return errors.New("bad request")
	}
	log.Printf("Got %s path\n>>>\n%s\n<<<", path, strings.TrimSpace(string(dump)))
	fmt.Println()
	return nil
}
