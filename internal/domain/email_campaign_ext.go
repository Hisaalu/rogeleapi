package domain

type EmailRecipient struct {
	ID        int64   `json:"id"`
	Email     string  `json:"email"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

func (r *EmailRecipient) FullName() string {
	fn, ln := "", ""
	if r.FirstName != nil {
		fn = *r.FirstName
	}
	if r.LastName != nil {
		ln = *r.LastName
	}
	switch {
	case fn == "" && ln == "":
		return ""
	case fn == "":
		return ln
	case ln == "":
		return fn
	}
	return fn + " " + ln
}

func (r *EmailRecipient) DisplayName() string {
	if name := r.FullName(); name != "" {
		return name
	}
	return r.Email
}
