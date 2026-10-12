package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const esquemaRetiradaCartera = "vec.meritos.hecho.retirada.v1"

var claveRetiradaCartera = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

type falloRetiradaCartera struct{ causa error }

func (falloRetiradaCartera) Error() string { return ports.ErrCarteraNoDisponible.Error() }
func (e falloRetiradaCartera) Unwrap() []error {
	return []error{ports.ErrCarteraNoDisponible, e.causa}
}

func errorInternoRetirada(causa error) error {
	if causa == nil {
		return ports.ErrCarteraNoDisponible
	}
	return falloRetiradaCartera{causa: causa}
}

type SolicitudRetiradaCartera struct {
	ResultadoContexto             vec.ResultadoContextoActorRegistradoV2
	Vinculo                       vec.VinculoAutenticacionActorV2
	Correlacion                   vec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion            vec.ReferenciaEntradaCatalogo
	HechoRef                      string
	HechoVersionEsperada          int
	DisponibilidadVersionEsperada uint64
	MotivoRetirada                vec.ReferenciaEntradaCatalogo
	ClaveIdempotencia             string
}

type ServicioRetiradaCartera struct {
	repositorio ports.RepositorioRetiradaCartera
	auditor     ports.AuditorIntentosRetiradaCartera
	autorizador vecports.AutorizadorSolicitudLigadaV3
	proveedor   ports.ProveedorMaterialRetiradaCartera
	reloj       vecports.Reloj
}

func NuevoServicioRetiradaCartera(repositorio ports.RepositorioRetiradaCartera,
	auditor ports.AuditorIntentosRetiradaCartera,
	autorizador vecports.AutorizadorSolicitudLigadaV3,
	proveedor ports.ProveedorMaterialRetiradaCartera,
	reloj vecports.Reloj) (*ServicioRetiradaCartera, error) {
	if nulo(repositorio) || nulo(auditor) || nulo(autorizador) || nulo(proveedor) || nulo(reloj) {
		return nil, ports.ErrCarteraNoDisponible
	}
	return &ServicioRetiradaCartera{repositorio, auditor, autorizador, proveedor, reloj}, nil
}

// Retirar exige una decisión V3 nueva incluso al recuperar la misma clave. El
// repositorio compara bytes originales antes del CAS y devuelve el recibo
// histórico solo tras auditar el nuevo acceso en su propia transacción.
func (s *ServicioRetiradaCartera) Retirar(ctx context.Context,
	q SolicitudRetiradaCartera) (ports.ConfirmacionRetiradaCartera, error) {
	vacio := ports.ConfirmacionRetiradaCartera{}
	if ctx == nil || s == nil || nulo(s.repositorio) || nulo(s.auditor) || nulo(s.autorizador) ||
		nulo(s.proveedor) || nulo(s.reloj) {
		return vacio, ports.ErrCarteraNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	datos, err := q.Vinculo.Datos()
	if err != nil || q.ResultadoContexto.Validar() != nil ||
		q.Vinculo.ValidarPara(q.ResultadoContexto) != nil || !q.Vinculo.VigenteEn(ahora, q.ResultadoContexto) ||
		q.Correlacion.Validar() != nil || datos.Superficie != vec.SuperficieAutenticacionExternaPersonalV1 ||
		datos.GarantiaObservada != vec.AuthAssuranceHigh || datos.CuentaPrivilegiada ||
		datos.PrincipalID != q.ResultadoContexto.Contexto.PersonaRef ||
		!domain.ReferenciaValida(datos.PrincipalID) {
		return vacio, ports.ErrCarteraDenegada
	}
	correlacion, err := q.Correlacion.ValorCanonico()
	if err != nil {
		return vacio, ports.ErrCarteraDenegada
	}
	if !domain.ReferenciaValida(q.HechoRef) || q.HechoVersionEsperada < 1 ||
		q.HechoVersionEsperada > 1073741824 || q.DisponibilidadVersionEsperada != 0 ||
		!claveRetiradaCartera.MatchString(q.ClaveIdempotencia) ||
		q.MotivoRetirada.Validar() != nil {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "denegado")
	}
	if !vec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	comando, err := canonRetiradaCartera(q, datos.PrincipalID)
	if err != nil {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	h := sha256.Sum256(comando)
	huella := hex.EncodeToString(h[:])
	recurso := vec.RecursoAutorizable{Referencia: q.HechoRef, ModuloID: ports.ModuloCartera,
		Tipo: "hecho", Ambitos: map[string]string{"persona_ref": datos.PrincipalID,
			"hecho_version_esperada":          strconv.Itoa(q.HechoVersionEsperada),
			"disponibilidad_version_esperada": strconv.FormatUint(q.DisponibilidadVersionEsperada, 10),
			"huella_comando_sha256":           huella},
		Atributos: map[string]string{"propiedad": "persona"}}
	if recurso.Validar() != nil {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	nominal, err := vec.NuevaSolicitudAutorizacionLigadaV3(vec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion,
		Accion: ports.AccionRetirarCartera, Recurso: recurso,
		Finalidad: ports.FinalidadCartera, Correlacion: q.Correlacion,
	})
	if err != nil {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	decision, confirmacion, err := s.autorizador.ExigirSolicitudLigadaV3(ctx, nominal, q.ResultadoContexto)
	if err != nil {
		if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacio, ports.ErrCarteraDenegada
		}
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	if decision.ExigirProyeccionPara(nominal, []string{"disponibilidad", "recibo"}, []string{"auditar"}) != nil {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	exportador, err := s.proveedor.EmitirMaterialRetiradaCartera(ctx, nominal,
		q.ResultadoContexto, decision, confirmacion)
	if err != nil || nulo(exportador) {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(nominal, decision, confirmacion,
		q.ResultadoContexto, q.MotivoAutorizacion, material, ports.AudienciaRetirarCartera) {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	ahora = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	resumen := material.ResumenCapacidad()
	if ahora.IsZero() || ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) ||
		!q.Vinculo.VigenteEn(ahora, q.ResultadoContexto) {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	resultado, err := s.repositorio.RetirarCartera(ctx, ports.OrdenRetiradaCartera{
		PersonaRef: datos.PrincipalID, HechoRef: q.HechoRef,
		HechoVersionEsperada:          q.HechoVersionEsperada,
		DisponibilidadVersionEsperada: q.DisponibilidadVersionEsperada,
		ClaveIdempotencia:             q.ClaveIdempotencia, HuellaComandoSHA256: huella,
		ComandoCanonico: comando, Material: material,
	})
	if err != nil {
		return vacio, s.registrarFalloRetirada(ctx, q, datos, correlacion, "error")
	}
	if resultado.Codigo == "conflicto" && resultado.Recibo == nil && resultado.AuditoriaRef != "" {
		return vacio, ports.ErrCarteraConflicto
	}
	if resultado.Codigo != "confirmada" && resultado.Codigo != "repetida" ||
		resultado.Recibo == nil || resultado.AuditoriaRef == "" {
		return vacio, ports.ErrCarteraNoDisponible
	}
	r := *resultado.Recibo
	if !domain.ReferenciaValida(r.Referencia) || r.HechoRef != q.HechoRef ||
		r.VersionHecho != q.HechoVersionEsperada || r.Disponibilidad.Estado != "retirado" ||
		r.Disponibilidad.Version != q.DisponibilidadVersionEsperada+1 ||
		r.RetiradoEn.IsZero() || r.RetiradoEn.After(s.reloj.Ahora().UTC().Add(time.Second)) ||
		!domain.ReferenciaValida(r.EventoRef) ||
		!domain.ReferenciaValida(r.AuditoriaRef) ||
		r.ClaveIdempotencia != q.ClaveIdempotencia || r.HuellaComandoSHA256 != huella {
		return vacio, ports.ErrCarteraNoDisponible
	}
	return ports.ConfirmacionRetiradaCartera{Recibo: r,
		Repetida: resultado.Codigo == "repetida", AuditoriaAccesoRef: resultado.AuditoriaRef}, nil
}

func (s *ServicioRetiradaCartera) registrarFalloRetirada(ctx context.Context,
	q SolicitudRetiradaCartera, datos vec.DatosVinculoAutenticacionActorV2,
	correlacion, resultado string) error {
	identidad := SolicitudCartera{ResultadoContexto: q.ResultadoContexto,
		Vinculo: q.Vinculo, Correlacion: q.Correlacion}
	base, contexto, vinculo, err := capturaBaseCartera(identidad, datos, correlacion, true)
	if err != nil {
		return ports.ErrCarteraNoDisponible
	}
	base["accion"] = ports.AccionRetirarCartera
	base["resultado"] = resultado
	base["motivo_ref"] = "intento_retirada_cartera_meritos_" + resultado
	recurso := sha256.Sum256([]byte(datos.PrincipalID))
	base["recurso_ref"] = "cartera:" + hex.EncodeToString(recurso[:])
	b, err := json.Marshal(base)
	if err != nil {
		return ports.ErrCarteraNoDisponible
	}
	captura, err := ports.NuevaCapturaLecturaCartera(datos.PrincipalID, correlacion,
		contexto, vinculo, b, datos.SesionValidaHasta)
	if err != nil {
		return ports.ErrCarteraNoDisponible
	}
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancelar()
	if err := s.auditor.RegistrarIntentoRetiradaCartera(auditCtx, captura); err != nil {
		return errorInternoRetirada(err)
	}
	if resultado == "denegado" {
		return ports.ErrCarteraEntradaInvalida
	}
	return ports.ErrCarteraNoDisponible
}

func canonRetiradaCartera(q SolicitudRetiradaCartera, persona string) ([]byte, error) {
	return json.Marshal(struct {
		Esquema                       string                        `json:"esquema"`
		ActorRef                      string                        `json:"actor_ref"`
		PersonaRef                    string                        `json:"persona_ref"`
		HechoRef                      string                        `json:"hecho_ref"`
		HechoVersionEsperada          int                           `json:"hecho_version_esperada"`
		DisponibilidadVersionEsperada uint64                        `json:"disponibilidad_version_esperada"`
		Motivo                        vec.ReferenciaEntradaCatalogo `json:"motivo"`
		ClaveIdempotencia             string                        `json:"clave_idempotencia"`
	}{esquemaRetiradaCartera, persona, persona, q.HechoRef, q.HechoVersionEsperada,
		q.DisponibilidadVersionEsperada, q.MotivoRetirada, q.ClaveIdempotencia})
}
