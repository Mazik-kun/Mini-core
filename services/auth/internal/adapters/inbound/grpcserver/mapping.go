package grpcserver

import (
	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/pkg/roles"
)

func roleToProto(role string) authv1.Role {
    switch role {
    case roles.RoleClient:
        return authv1.Role_ROLE_CLIENT
    case roles.RoleOfficer:
        return authv1.Role_ROLE_OFFICER
    case roles.RoleAdmin:
        return authv1.Role_ROLE_ADMIN
    default:
        return authv1.Role_ROLE_UNSPECIFIED
    }
}

func roleFromProto(r authv1.Role) string {
    switch r {
    case authv1.Role_ROLE_CLIENT:
        return roles.RoleClient
    case authv1.Role_ROLE_OFFICER:
        return roles.RoleOfficer
    case authv1.Role_ROLE_ADMIN:
        return roles.RoleAdmin
    default:
        return ""
    }
}