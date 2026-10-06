package domain

type Status string

const (
	StatusNew = "NEW"
	StatusProfileFilled = "PROFILE_FILLED"
	StatusOnKYC = "ON_KYC"
	StatusActive = "ACTIVE"
	StatusRejected =  "REJECTED"
	StatusBlocked = "BLOCKED"
)

func (s Status)CanTransitionTo(target Status) bool{
	switch s {
	case StatusNew:
		if target == StatusProfileFilled{
			return true
		}
	case StatusProfileFilled:
		if target == StatusOnKYC{
			return true
		}
	case StatusOnKYC:
		if target == StatusActive || target == StatusRejected{
			return true
		}
	case StatusActive:
		if target == StatusBlocked{
			return true
		}
	case StatusBlocked:
		if target == StatusActive{
			return true
		}
	}
	return false
}

func(s Status)IsValid() bool{
	if s == StatusNew || s == StatusOnKYC || s == StatusActive || s == StatusProfileFilled || s == StatusRejected || s == StatusBlocked{
		return true
	}
	return false
}

func (s Status)CanEditProfile()bool{
	if s == StatusNew || s == StatusProfileFilled{
		return true
	}
	return false
}