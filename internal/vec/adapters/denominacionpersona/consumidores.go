package denominacionpersona

import (
	"context"
	"sync"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type Preparador struct {
	protector ports.ProtectorDenominacionPersona
	personas  ports.ExistenciaPersonaDenominacion
}

func NuevoPreparador(protector ports.ProtectorDenominacionPersona, personas ports.ExistenciaPersonaDenominacion) (*Preparador, error) {
	if nulo(protector) || nulo(personas) {
		return nil, ErrNoDisponible
	}
	return &Preparador{protector: protector, personas: personas}, nil
}

func (p *Preparador) PrepararDenominacionPersona(ctx context.Context, persona string, esperada uint64, procedencia, ambito, claveIndiceRef string, nombre []byte) (ports.PreparacionDenominacionPersona, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || !domain.ReferenciaPersonaDenominacionValida(persona) || p.personas.RevalidarPersonaDenominacion(ctx, persona) != nil {
		return ports.PreparacionDenominacionPersona{}, ErrNoDisponible
	}
	prep, err := p.protector.PrepararDenominacionPersona(ctx, persona, esperada, procedencia, ambito, claveIndiceRef, nombre)
	if err != nil || ctx.Err() != nil || p.personas.RevalidarPersonaDenominacion(ctx, persona) != nil {
		return ports.PreparacionDenominacionPersona{}, ErrNoDisponible
	}
	return prep, nil
}

type Lector struct {
	protector     ports.ProtectorDenominacionPersona
	personas      ports.ExistenciaPersonaDenominacion
	fuente        ports.FuenteDenominacionPersonaAutorizada
	intentos      ports.RegistradorIntentosAuditoria
	configuracion ConfiguracionIntentos
}

func NuevoLector(protector ports.ProtectorDenominacionPersona, personas ports.ExistenciaPersonaDenominacion, fuente ports.FuenteDenominacionPersonaAutorizada, intentos ports.RegistradorIntentosAuditoria, configuracion ConfiguracionIntentos) (*Lector, error) {
	if nulo(protector) || nulo(personas) || nulo(fuente) || nulo(intentos) || configuracion.MotivoError.Validar() != nil || configuracion.Proceso == "" || configuracion.Canal == "" {
		return nil, ErrNoDisponible
	}
	return &Lector{protector: protector, personas: personas, fuente: fuente, intentos: intentos, configuracion: configuracion}, nil
}

func clonarSobre(s ports.SobreDenominacionPersona) ports.SobreDenominacionPersona {
	s.Nonce = append([]byte(nil), s.Nonce...)
	s.Cifrado = append([]byte(nil), s.Cifrado...)
	ts := make([][]byte, len(s.Indice.Tokens))
	for i, t := range s.Indice.Tokens {
		ts[i] = append([]byte(nil), t...)
	}
	s.Indice.Tokens = ts
	return s
}

// El puerto autorizado confirma primero consumo/auditoría. Un COMMIT fallido
// impide llegar al KMS y al callback. Antes de prestar el nombre vuelve a
// comprobar Persona y vigencia con el mismo puerto, nunca con permisos de correo.
func (l *Lector) ConDenominacionPersona(ctx context.Context, acceso ports.AccesoDenominacionPersona, usar func(domain.DenominacionPersona) error) error {
	if l == nil || !l.evidenciaValida(acceso, ports.AccionLeerDenominacionPersona) {
		return ErrNoDisponible
	}
	acceso = clonarAcceso(acceso)
	err := l.leer(ctx, acceso, usar)
	if err != nil {
		return l.registrarError(ctx, acceso, ports.AccionLeerDenominacionPersona)
	}
	return nil
}

func (l *Lector) leer(ctx context.Context, acceso ports.AccesoDenominacionPersona, usar func(domain.DenominacionPersona) error) error {
	if l == nil || ctx == nil || ctx.Err() != nil || usar == nil || !domain.ReferenciaPersonaDenominacionValida(acceso.PersonaRef) || acceso.Version == 0 || acceso.Version > 1<<53-1 {
		return ErrNoDisponible
	}
	lectura, err := l.fuente.LeerDenominacionPersonaAutorizada(ctx, clonarAcceso(acceso))
	sobre := lectura.Sobre
	acuse, errAcuse := lectura.Acuse.Datos()
	if err != nil || ctx.Err() != nil || errAcuse != nil || sobre.PersonaRef != acceso.PersonaRef || sobre.Version != acceso.Version || acuse.PersonaRef != acceso.PersonaRef || acuse.Version != acceso.Version || acuse.SobreSHA256 != huellaSobre(sobre) || acuse.RecursoRef != acceso.Recurso.Referencia || acuse.CorrelacionRef != acceso.Auditoria.CorrelationRef || l.fuente.ValidarAcuseDenominacionPersona(ctx, clonarAcceso(acceso), lectura) != nil {
		return ErrNoDisponible
	}
	sobre = clonarSobre(sobre)
	var mu sync.Mutex
	abierto, invocado := true, false
	var callbackErr error
	err = l.protector.ConDenominacionDescifrada(ctx, clonarSobre(sobre), func(d domain.DenominacionPersona) error {
		mu.Lock()
		defer mu.Unlock()
		if !abierto || invocado {
			return ErrNoDisponible
		}
		invocado = true
		if ctx.Err() != nil || d.PersonaRef() != acceso.PersonaRef || d.Version() != acceso.Version || l.personas.RevalidarPersonaDenominacion(ctx, acceso.PersonaRef) != nil || l.fuente.RevalidarAccesoDenominacionPersona(ctx, clonarAcceso(acceso), clonarSobre(sobre)) != nil || l.protector.RevalidarProteccionDenominacionPersona(ctx, clonarSobre(sobre)) != nil || ctx.Err() != nil {
			callbackErr = ErrNoDisponible
			return callbackErr
		}
		callbackErr = usar(d)
		return callbackErr
	})
	mu.Lock()
	abierto = false
	ejecutado, fallo := invocado, callbackErr
	mu.Unlock()
	if fallo != nil {
		return fallo
	}
	if err != nil || !ejecutado {
		return ErrNoDisponible
	}
	return nil
}

// La búsqueda delega la selección a la autoridad del índice antes de paginar.
// Su resultado sólo contiene referencias; cada nombre exige lectura propia.
func (l *Lector) BuscarDenominacionPersona(ctx context.Context, acceso ports.AccesoDenominacionPersona, ambito, claveIndiceRef string, terminos []byte, limite int) ([]ports.ReferenciaDenominacionPersona, error) {
	if l == nil || !l.evidenciaValida(acceso, ports.AccionBuscarDenominacionPersona) {
		return nil, ErrNoDisponible
	}
	acceso = clonarAcceso(acceso)
	refs, err := l.buscar(ctx, acceso, ambito, claveIndiceRef, terminos, limite)
	if err != nil {
		return nil, l.registrarError(ctx, acceso, ports.AccionBuscarDenominacionPersona)
	}
	return refs, nil
}

func (l *Lector) buscar(ctx context.Context, acceso ports.AccesoDenominacionPersona, ambito, claveIndiceRef string, terminos []byte, limite int) ([]ports.ReferenciaDenominacionPersona, error) {
	if l == nil || ctx == nil || ctx.Err() != nil || limite < 1 || limite > 100 {
		return nil, ErrNoDisponible
	}
	i, err := l.protector.PrepararBusquedaDenominacionPersona(ctx, ambito, claveIndiceRef, terminos)
	if err != nil {
		return nil, ErrNoDisponible
	}
	refs, err := l.fuente.BuscarDenominacionPersonaAutorizada(ctx, acceso, i, limite)
	if err != nil || ctx.Err() != nil || len(refs) > limite {
		return nil, ErrNoDisponible
	}
	vistos := map[string]bool{}
	for _, ref := range refs {
		if !domain.ReferenciaPersonaDenominacionValida(ref.PersonaRef) || ref.Version == 0 || ref.Version > 1<<53-1 || vistos[ref.PersonaRef] {
			return nil, ErrNoDisponible
		}
		vistos[ref.PersonaRef] = true
	}
	return append([]ports.ReferenciaDenominacionPersona(nil), refs...), nil
}
