package member

import (
	"database/sql"
	"github.com/lib/pq"
	"log"
)

type Admin struct {
	privileges []string
}

func (m *Member) get_admin() {
	var privileges pq.StringArray
	if err := m.QueryRow(
		"SELECT privileges "+
			"FROM administrator "+
			"WHERE member = $1", m.Id).
		Scan(&privileges); err != nil {
		if err != sql.ErrNoRows {
			log.Panic(err)
		}
		return
	}
	m.Admin = &Admin{privileges}
}

func (m *Member) set_gratuitous() {
	m.Gratuitous = true
	if _, err := m.Exec(
		"UPDATE member "+
		"SET gratuitous = 't' "+
		"WHERE id = $1", m.Id); err != nil {
		log.Panic(err)
	}
}

// Approve_member sets the approval flag on <m> and activates the invoice if
//	m.Membership_invoice exists, otherwise setting the gratuitous flag.
//BUG: approving a member with unverified e-mail will leave the member out of the "Members" talk group, requiring manual intervention
func (a *Member) Approve_member(m *Member) {
	if a.Admin == nil {
		log.Panicf("%s is not an administrator\n", a.Username)
	}
	if m.Approved {
		log.Panicf("%s is already an approved member\n", m.Username)
	}
	if _, err := m.Exec(
		"UPDATE member "+
		"SET"+
		"	approved_at = now(),"+
		"	approved_by = $1 ", a.Id); err != nil {
		log.Panic(err);
	}
	m.Approved = true
	if m.Talk_user() != nil {
		m.talk.Add_to_group("Members")
	}
	if m.Membership_invoice != nil {
		m.Payment().Approve_pending_membership(m.Membership_invoice)
	} else {
		m.set_gratuitous()
	}
}

