package site

import (
	"time"
)

func init() {
	init_handler("billing", billing_handler, "/member/billing")
}

func billing_handler(p *page) {
	p.Title = "Billing"
	if !p.must_authenticate() {
		return
	}
	if token := p.PostFormValue("stripeToken"); token != "" {
		p.Data["card_error"] = p.Update_customer(token, nil)
		return
	}
	if subitem_id := p.PostFormValue("cancel-membership"); subitem_id != "" {
		mp := p.Member.Get_membership()
		if subitem_id != mp.ID {
			p.http_error(400)
			return
		}
		if !p.Member.Authenticate(p.PostFormValue("password")) {
			p.Data["password_error"] = "Incorrect password"
			return
		}
		//TODO: reason for cancellation: PostFormValue("cancellation_reason")
		p.Member.Cancel_membership()
		p.redirect = "/member/billing"
	} else if _, ok := p.PostForm["register-membership"]; ok {
		rate := p.PostFormValue("rate")
		if rate == "membership-student" &&
			p.PostFormValue("institution") != "" &&
			p.PostFormValue("student-email") != "" &&
			p.PostFormValue("graduation-date") != "" {
			//TODO institution/email verification
			graduation, err := time.Parse("2006-01",
				p.PostFormValue("graduation-date"))
			if err != nil {
				p.http_error(400)
				return
			}
			if err := p.Update_student(p.PostFormValue("institution"),
				p.PostFormValue("student-email"), graduation);
				err != nil {
				p.Data["membership_registration_error"] = err
				return
			}
		}
		membership := p.Member.Get_membership()
		if rate == "" || membership != nil && rate == membership.Plan.ID {
			p.Data["membership_registration_error"] = "Already registered for "+
				membership.Plan.Name
			return
		}
		if err := p.Request_membership(rate); err != nil {
			p.Data["membership_registration_error"] = err
			return
		}
		p.redirect = "/member/billing"
	} else if _, ok := p.PostForm["cancel-pending-membership"]; ok {
		pending := p.Member.Get_pending_membership()
		if pending == nil {
			p.http_error(400)
			return
		}
		p.Cancel_pending_subscription(pending)
	} else if subitem_id := p.PostFormValue("cancel-subscription");
		subitem_id != "" {
		if subitem_id == p.Membership_id() {
			p.http_error(403)
			return
		}
		s, err := p.Get_subscription_from_item(subitem_id)
		if err != nil {
			p.Data["cancel_subscription_error"] = err
			return
		}
		if err := p.Cancel_subscription_item(s.ID, subitem_id); err != nil {
			p.Data["cancel_subscription_error"] = err
			return
		}
		p.redirect = "/member/billing"
	}
}
