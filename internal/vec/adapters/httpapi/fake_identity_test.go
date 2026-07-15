package httpapi

import (
	"context"
	"errors"
	"net"
	"net/http"

	"vec-diputacion-granada/internal/vec/domain"
)

// resolvedorIdentidadPruebas hace explicita la identidad que los tests de la
// carcasa necesitan. No forma parte de la composicion ejecutable.
type resolvedorIdentidadPruebas struct{}

func (resolvedorIdentidadPruebas) ResolveDemoIdentity(_ context.Context, peticion *http.Request) (domain.Principal, error) {
	_, todasIPv4, _ := net.ParseCIDR("0.0.0.0/0")
	_, todasIPv6, _ := net.ParseCIDR("::/0")
	politica := identityPolicy{
		trustHeaders:    true,
		trustedProxies:  []*net.IPNet{todasIPv4, todasIPv6},
		subjectHeader:   "X-VEC-Subject",
		rolesHeader:     "X-VEC-Roles",
		mechanismHeader: "X-VEC-Auth-Mechanism",
	}
	identidad := identityFromRequest(peticion, politica)
	hayAsercion := peticion != nil && (peticion.TLS != nil ||
		peticion.Header.Get("X-Auth-Subject") != "" || peticion.Header.Get("X-VEC-Subject") != "")
	if identidad.subject == "" && hayAsercion {
		return domain.Principal{}, errors.New("identidad de prueba no valida")
	}
	if identidad.subject == "" {
		identidad = requestIdentity{
			subject:     "actor-prueba-explicito",
			displayName: "Actor de prueba explicito",
			method:      domain.AuthMethodDemo,
			assurance:   domain.AuthAssuranceHigh,
			roles:       []string{"administrador"},
			attributes:  map[string]string{},
		}
	}
	// Este resolvedor solo existe en pruebas. Las garantias y roles abreviados
	// se fijan aqui de forma deliberada para los fixtures antiguos; la frontera
	// HTTP productiva no los deduce del nombre del mecanismo ni del sujeto.
	if identidad.subject != "" && !identidad.assurance.Valida() {
		identidad.assurance = domain.AuthAssuranceHigh
	}
	if len(identidad.roles) == 0 && identidad.subject == "candidate" {
		identidad.roles = []string{"ciudadano"}
	}
	if !identidad.method.Valido() && identidad.subject == "staff" {
		identidad.method = domain.AuthMethodDemo
		identidad.assurance = domain.AuthAssuranceHigh
		identidad.roles = []string{"administrador"}
	}
	principal := domain.Principal{
		ID:            identidad.subject,
		DisplayName:   identidad.displayName,
		Email:         identidad.email,
		Roles:         append([]string(nil), identidad.roles...),
		Permissions:   []string{},
		AuthMethod:    identidad.method,
		AuthAssurance: identidad.assurance,
		Attributes:    identidad.attributes,
	}
	if err := principal.Validate(); err != nil {
		return domain.Principal{}, err
	}
	return principal, nil
}
