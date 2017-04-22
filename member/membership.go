package member

import (
	"fmt"
	"strings"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/subitem"
	"log"
)

func (m *Member) Request_membership(rate string) error {
	p, ok := m.Plans["membership-" + rate]
	if !ok {
		return fmt.Errorf("Invalid membership rate")
	}
	if rate == "student" && m.Student == nil {
		return fmt.Errorf("Non-students cannot apply for a student membership "+
			"rate")
	}
	if mp := m.Get_membership(); mp != nil {
		if mp.Plan.ID == p.ID {
			return fmt.Errorf("Cannot duplicate existing membership")
		}
		if mp.Plan.Amount < p.Amount {
			return m.Update_membership("membership-" + rate)
		}
	}
	return m.Request_subscription("membership-" + rate)
}

func (m *Member) Get_pending_membership() *Pending_subscription {
	for _, ps := range m.Get_pending_subscriptions() {
		if strings.HasPrefix(ps.Plan_id, "membership-") {
			return ps
		}
	}
	return nil
}

func (m *Member) Get_membership() *stripe.SubItem {
	if m.Customer() == nil {
		return nil
	}
	for _, s := range m.customer.Subscriptions {
		for _, i := range s.Items.Values {
			if Plan_category(i.Plan.ID) == "membership" {
				return i
			}
		}
	}
	return nil
}

func (m *Member) Membership_rate() string {
	if ms := m.Get_membership(); ms != nil {
		return Plan_identifier(ms.Plan.ID)
	}
	return ""
}

func (m *Member) Membership_id() string {
	if ms := m.Get_membership(); ms != nil {
		return ms.ID
	}
	return ""
}

func (m *Member) Update_membership(plan_base_id string) error {
	p, ok := m.Plans[plan_base_id]
	if !ok {
		return fmt.Errorf("Invalid plan '%s'", plan_base_id)
	}
	mp := m.Get_membership()
	if mp == nil {
		return fmt.Errorf("No existing membership")
	}
	_, err := subitem.Update(mp.ID, &stripe.SubItemParams{Plan: p.ID})
	return err
}

func (m *Member) Cancel_membership() {
	mp := m.Get_membership()
	if mp != nil {
		s, err := m.Get_subscription_from_item(mp.ID)
		if err != nil {
			log.Println(err)
		} else if err = m.Cancel_subscription_item(s.ID, mp.ID); err != nil {
			log.Println(err)
		}
	}
	if m.Talk_user() != nil {
		m.Talk_user().Remove_from_group("Members")
	}
}
