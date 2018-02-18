package talk

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

/*
func (api *Api) Check_username(username string) (available bool, err string) {
	j, e := api.get_json("/users/check_username.json?username="+
		url.QueryEscape(username), false)
	if e != nil {
		log.Println(e)
		return false, "Server error"
	}
	if j, ok := j.(map[string]interface{}); ok {
		if errors, ok := j["errors"]; ok {
			return false, "Username " + errors.([]interface{})[0].(string)
		} else if _, ok := j["available"]; ok {
			// Even if talk gives available = false, (e.g. for staged users),
			//	sso automatically merges the talk user instance with the
			//	newly-created local one, hence we still return 'true'
			return true, ""
		}
		// Unanticipated JSON response
		log.Printf("Talk server rejected username '%s': %q\n", username, j)
		return false, "Username not available"
	}
	log.Panicf("Talk server parsing error during Check_username: %s\n",
		username)
	return
}*/

type User struct {
	External_id    int
	Id             int
	Username       string
	Admin          bool
	Avatar_tmpl    string
	Title          string
	Groups         map[string]int
	Bio            string
	Website_url    string
	Website_name   string
	Location       string
	Card_bg_url    string
	Profile_bg_url string
	*Api
}

func (api *Api) parse_user(u map[string]interface{}) *User {
	t := &User{Api: api}
	t.Id = int(u["id"].(float64))
	t.Username = u["username"].(string)
	t.Admin = u["admin"].(bool)
	t.Avatar_tmpl = u["avatar_template"].(string)
	t.Title, _ = u["title"].(string)
	t.Groups = make(map[string]int)
	for _, group := range u["groups"].([]interface{}) {
		g := group.(map[string]interface{})
		t.Groups[g["name"].(string)] = int(g["id"].(float64))
	}
	t.Bio, _ = u["bio_cooked"].(string)
	t.Website_url, _ = u["website"].(string)
	t.Website_name, _ = u["website_name"].(string)
	t.Location, _ = u["location"].(string)
	t.Card_bg_url, _ = u["card_background"].(string)
	t.Profile_bg_url, _ = u["profile_background"].(string)
	if sso, ok := u["single_sign_on_record"].(map[string]interface{}); ok {
		if external_id, ok := sso["external_id"].(string); ok {
			t.External_id, _ = strconv.Atoi(external_id)
		}
	}
	return t
}

func (api *Api) Get_user(id int) (*User, error) {
	j, err := api.get_json("/admin/users/"+fmt.Sprint(id)+".json", true)
	if err != nil {
		log.Printf("Talk error fetching user <%d>: %q", id, err)
		return nil, fmt.Errorf("Server error")
	}
	if j, ok := j.(map[string]interface{}); ok {
		if user := api.parse_user(j); user != nil {
			return user, nil
		}
	}
	log.Printf("Talk error fetching user <%d>: %q", id, err)
	return nil, fmt.Errorf("Invalid Talk user")
}

func (api *Api) Get_user_by_external_id(external_id int) (*User, error) {
	j, err := api.get_json("/users/by-external/"+fmt.Sprint(external_id)+
		".json", true)
	if err != nil {
		log.Println("Talk error fetching user by external id <%d>: %q",
			external_id, err)
		return nil, fmt.Errorf("Server error")
	}
	if j, ok := j.(map[string]interface{}); ok {
		if u, ok := j["user"].(map[string]interface{}); ok {
			user := api.parse_user(u)
			if user != nil {
				user.External_id = external_id
				return user, nil
			}
		}
	}
	log.Printf("Talk error fetching user by external id <%d>: %q",
		external_id, err)
	return nil, fmt.Errorf("Invalid Talk user")
}

func (api *Api) Get_user_by_email(email string) (*User, error) {
	j, err := api.get_json("/admin/users.json?filter="+url.QueryEscape(email), true)
	if err != nil {
		log.Println("Talk error fetching user by e-mail <"+email+">:", err)
		return nil, fmt.Errorf("Server error")
	}
	if j, ok := j.([]interface{}); ok {
		if len(j) == 0 {
			return nil, fmt.Errorf("E-mail address not in use")
		}
		if user, ok := j[0].(map[string]interface{}); ok {
			if id, ok := user["id"].(float64); ok {
				return api.Get_user(int(id))
			}
		}
	}
	log.Println("Talk error fetching user <"+email+">:", j)
	return nil, fmt.Errorf("Invalid Talk user")
}

//TODO: grab external_id from posters
type Message struct {
	Url             string
	Title           string
	Read            bool
	Last_post       time.Time
	First_post      time.Time
	Reply_count     int
	Poster_avatars  map[string]string
	Last_poster     string
	Original_poster string
}

func (t *User) Get_messages(limit int) []*Message {
	msgs := make([]*Message, 0)
	usernames := make(map[int]string)
	avatars := make(map[int]string)
	j, _ := t.get_json("/topics/private-messages/"+t.Username+".json", true)
	if j, ok := j.(map[string]interface{}); ok {
		if u, ok := j["users"].([]interface{}); ok {
			for _, v := range u {
				user := v.(map[string]interface{})
				id := int(user["id"].(float64))
				usernames[id] = user["username"].(string)
				avatars[id] = strings.Replace(user["avatar_template"].(string),
					"{size}", "120", 1)
			}
		}
		if tp, ok := j["topic_list"].(map[string]interface{}); ok {
			if tp, ok := tp["topics"].([]interface{}); ok {
				var c int
				for _, v := range tp {
					msg := &Message{Read: true}
					topic := v.(map[string]interface{})
					id := int(topic["id"].(float64))
					slug := topic["slug"].(string)
					msg.Url = t.Path + "/t/" + slug + "/" + fmt.Sprint(id)
					msg.Title = topic["title"].(string)
					msg.Reply_count = int(topic["posts_count"].(float64)) - 1
					msg.First_post, _ = time.Parse("2006-01-02T15:04:05.999Z",
						topic["created_at"].(string))
					msg.Last_post, _ = time.Parse("2006-01-02T15:04:05.999Z",
						topic["last_posted_at"].(string))
					if topic["unseen"].(bool) == true {
						msg.Read = false
					} else if l := topic["highest_post_number"].(float64); l != 0 {
						msg.Url += "/" + fmt.Sprint(int(l))
					}
					msg.Last_poster = topic["last_poster_username"].(string)
					msg.Poster_avatars = make(map[string]string)
					for _, p := range topic["posters"].([]interface{}) {
						if p, ok := p.(map[string]interface{}); ok {
							i := int(p["user_id"].(float64))
							msg.Poster_avatars[usernames[i]] = avatars[i]
							if strings.Contains(p["description"].(string),
								"Original Poster") {
								msg.Original_poster = usernames[i]
							}
						}
					}
					msgs = append(msgs, msg)
					if c++; limit == c {
						break
					}
				}
			}
		}
	}
	return msgs
}

func (t *User) Add_to_group(group string) error {
	err := t.Api.Add_to_group(group, t)
	if err == nil {
		t.Groups[group] = t.All_groups()[group]
	}
	return err
}

func (t *User) Remove_from_group(group string) {
	//TODO propagate errors
	if _, ok := t.Groups[group]; !ok {
		// Not in group
		return
	}
	gid, ok := t.All_groups()[group]
	if !ok {
		log.Printf("'%s' is not a valid group.\n", group)
		return
	}
	form := url.Values{}
	form.Set("user_id", fmt.Sprint(t.Id))
	data, err := t.do_form("DELETE", "/groups/"+fmt.Sprint(gid)+"/members",
		form)
	if err != nil {
		log.Printf("Failed to remove %s from Talk group %s: %q", t.Username,
			group, err)
		return
	}
	if j, ok := data.(map[string]interface{}); ok {
		if _, ok := j["success"]; ok {
			delete(t.Groups, group)
			return
		}
	}
	log.Printf("Talk error on removing %s from group %s: %q\n",
		t.Username, group, data)
}
