package application

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"strconv"

	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrFuenteCargaCustodiaNoDisponible = errors.New("documentos: lectura de carga admitida no disponible")

type emisorNativoLecturaCargaAdmitida interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3,
		vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3,
		vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
		vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type generadorCorrelacionLecturaCargaAdmitida interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
}

// ServicioFuenteCargaAdmitidaCustodia implementa el puerto neutral Mayan.
// La primera autoridad consume AD240 antes de SELECT; la segunda, nativa de
// almacén, se obtiene para el objeto/version de esa fila antes de GET.
type ServicioFuenteCargaAdmitidaCustodia struct {
	repositorio   vecports.RepositorioConsultaCargaAdmitidaCustodiaV3
	autoridadSQL  vecports.AutoridadConsultaCargaAdmitidaCustodiaV3
	emisorAlmacen emisorNativoLecturaCargaAdmitida
	almacen       vecports.AlmacenObjetos
	seudonimos    vecports.SeudonimizadorSujetoAlmacen
	correlaciones generadorCorrelacionLecturaCargaAdmitida
	motivo        vecdomain.ReferenciaEntradaCatalogo
	reloj         vecports.Reloj
}

func NuevaFuenteCargaAdmitidaCustodia(
	repositorio vecports.RepositorioConsultaCargaAdmitidaCustodiaV3,
	autoridadSQL vecports.AutoridadConsultaCargaAdmitidaCustodiaV3,
	emisorAlmacen emisorNativoLecturaCargaAdmitida,
	almacen vecports.AlmacenObjetos,
	seudonimos vecports.SeudonimizadorSujetoAlmacen,
	correlaciones generadorCorrelacionLecturaCargaAdmitida,
	motivo vecdomain.ReferenciaEntradaCatalogo,
	reloj vecports.Reloj,
) (*ServicioFuenteCargaAdmitidaCustodia, error) {
	if dependenciaNula(repositorio) || dependenciaNula(autoridadSQL) || dependenciaNula(emisorAlmacen) ||
		dependenciaNula(almacen) || dependenciaNula(seudonimos) || dependenciaNula(correlaciones) ||
		dependenciaNula(reloj) || motivo.Validar() != nil {
		return nil, ErrFuenteCargaCustodiaNoDisponible
	}
	return &ServicioFuenteCargaAdmitidaCustodia{repositorio, autoridadSQL, emisorAlmacen,
		almacen, seudonimos, correlaciones, motivo, reloj}, nil
}

func (s *ServicioFuenteCargaAdmitidaCustodia) consultar(ctx context.Context,
	entrada ports.SolicitudCargaAdmitidaCustodia,
	esperada *ports.DatosCargaAdmitidaCustodia,
) (ports.DatosCargaAdmitidaCustodia, vecdomain.ReferenciaCorrelacionAutorizacionV2, error) {
	if s == nil || ctx == nil || entrada.Validar() != nil ||
		(entrada.Fase == ports.FaseAperturaCargaCustodia) != (esperada != nil) {
		return ports.DatosCargaAdmitidaCustodia{}, vecdomain.ReferenciaCorrelacionAutorizacionV2{}, ErrFuenteCargaCustodiaNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, s.correlaciones)
	if err != nil {
		return ports.DatosCargaAdmitidaCustodia{}, vecdomain.ReferenciaCorrelacionAutorizacionV2{}, ErrFuenteCargaCustodiaNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return ports.DatosCargaAdmitidaCustodia{}, vecdomain.ReferenciaCorrelacionAutorizacionV2{}, ErrFuenteCargaCustodiaNoDisponible
	}
	q := vecports.ConsultaCargaAdmitidaCustodiaV3{
		Fase: entrada.Fase, CargaID: entrada.CargaID, RecursoRef: entrada.RecursoRef,
		PersonaRef: entrada.PersonaRef, PlanRef: entrada.PlanRef, VersionEsperada: entrada.VersionEsperada,
		ActorRegistradoV2: entrada.ActorRegistradoV2, VinculoV2: entrada.VinculoV2,
		Finalidad: entrada.Finalidad, CorrelacionRef: correlacionRef,
	}
	if esperada != nil {
		c := esperada.Carga
		huella, err := c.HuellaSHA256()
		if err != nil || esperada.HuellaCargaSHA256 != huella || c.Validar() != nil ||
			c.Estado != vecdomain.EstadoCargaDocumentalAdmitida || c.Version != 5 ||
			c.ID != entrada.CargaID || c.PrincipalID != entrada.PersonaRef ||
			c.RecursoRef != entrada.RecursoRef || c.ContenidoAdmitido == nil || c.Analisis == nil ||
			c.Analisis.Estado != vecdomain.EstadoAnalisisCargaLimpio {
			return ports.DatosCargaAdmitidaCustodia{}, vecdomain.ReferenciaCorrelacionAutorizacionV2{}, vecdomain.ErrAutorizacionDenegada
		}
		q.HuellaCargaEsperadaSHA256 = huella
		q.ObjetoRef = c.ContenidoAdmitido.Referencia
		q.ObjetoVersion = c.ContenidoAdmitido.Version
		q.HuellaContenidoSHA256 = c.ContenidoAdmitido.HuellaSHA256
		q.Tamano = c.ContenidoAdmitido.Tamano
		q.MIME = c.ContenidoAdmitido.MIME
		q.EvidenciaAnalisisRef = c.Analisis.EvidenciaRef
	}
	pre, err := vecports.PreimagenConsultaCargaAdmitidaCustodiaV3(q)
	if err != nil {
		return ports.DatosCargaAdmitidaCustodia{}, vecdomain.ReferenciaCorrelacionAutorizacionV2{}, vecdomain.ErrAutorizacionDenegada
	}
	a, err := s.autoridadSQL.AutorizarConsultaCargaAdmitidaCustodia(ctx, pre,
		entrada.ActorRegistradoV2, entrada.CargaID, entrada.RecursoRef)
	if err != nil {
		return ports.DatosCargaAdmitidaCustodia{}, vecdomain.ReferenciaCorrelacionAutorizacionV2{}, vecdomain.ErrAutorizacionDenegada
	}
	q.AutorizacionV3 = a
	if q.Validar(s.reloj.Ahora().UTC()) != nil {
		return ports.DatosCargaAdmitidaCustodia{}, vecdomain.ReferenciaCorrelacionAutorizacionV2{}, vecdomain.ErrAutorizacionDenegada
	}
	c, err := s.repositorio.ConsultarAdmitidaCustodia(ctx, q)
	if err != nil || c.Validar() != nil || c.ID != entrada.CargaID ||
		c.RecursoRef != entrada.RecursoRef || c.PrincipalID != entrada.PersonaRef ||
		c.Estado != vecdomain.EstadoCargaDocumentalAdmitida || c.Version != 5 ||
		c.ModuloID != "meritos" || c.Finalidad != entrada.Finalidad ||
		c.Analisis == nil || c.Analisis.Estado != vecdomain.EstadoAnalisisCargaLimpio ||
		c.ContenidoAdmitido == nil {
		return ports.DatosCargaAdmitidaCustodia{}, vecdomain.ReferenciaCorrelacionAutorizacionV2{}, ErrFuenteCargaCustodiaNoDisponible
	}
	huella, err := c.HuellaSHA256()
	if err != nil || (esperada != nil && huella != esperada.HuellaCargaSHA256) {
		return ports.DatosCargaAdmitidaCustodia{}, vecdomain.ReferenciaCorrelacionAutorizacionV2{}, vecdomain.ErrAutorizacionDenegada
	}
	return ports.DatosCargaAdmitidaCustodia{Carga: c, HuellaCargaSHA256: huella}, correlacion, nil
}

func (s *ServicioFuenteCargaAdmitidaCustodia) ConsultarAdmitida(ctx context.Context,
	entrada ports.SolicitudCargaAdmitidaCustodia) (ports.DatosCargaAdmitidaCustodia, error) {
	datos, _, err := s.consultar(ctx, entrada, nil)
	return datos, err
}

func (s *ServicioFuenteCargaAdmitidaCustodia) AbrirOriginalExacto(ctx context.Context,
	entrada ports.SolicitudCargaAdmitidaCustodia,
	esperada ports.DatosCargaAdmitidaCustodia,
) (ports.LecturaCargaAdmitidaCustodia, error) {
	datos, correlacion, err := s.consultar(ctx, entrada, &esperada)
	if err != nil {
		return ports.LecturaCargaAdmitidaCustodia{}, err
	}
	c := datos.Carga
	objeto := vecports.ReferenciaObjetoAlmacen{Referencia: c.ContenidoAdmitido.Referencia,
		Version: c.ContenidoAdmitido.Version}
	solicitudSeudonimo, err := vecports.NuevaSolicitudSeudonimizarSujetoAlmacen(c.PrincipalID, c.RecursoRef)
	if err != nil {
		return ports.LecturaCargaAdmitidaCustodia{}, ErrFuenteCargaCustodiaNoDisponible
	}
	seudonimo, err := s.seudonimos.SeudonimizarSujetoAlmacen(ctx, solicitudSeudonimo)
	if err != nil {
		return ports.LecturaCargaAdmitidaCustodia{}, ErrFuenteCargaCustodiaNoDisponible
	}
	vinculos := vecports.VinculosOperacionAlmacen{OperacionRef: entrada.PlanRef,
		CargaRef: c.ID, Clasificacion: c.Clasificacion, SujetoSeudonimoHMAC: seudonimo,
		HuellaSolicitudHMAC: c.HuellaSolicitudHMAC, EfectoRef: c.ContenidoAdmitido.Referencia,
		ObjetoVinculado: objeto}
	atributos := map[string]string{
		vecports.AtributoAlmacenOperacionRef:        entrada.PlanRef,
		vecports.AtributoAlmacenCargaRef:            c.ID,
		vecports.AtributoAlmacenClasificacion:       c.Clasificacion,
		vecports.AtributoAlmacenSujetoSeudonimoHMAC: seudonimo,
		vecports.AtributoAlmacenHuellaSolicitudHMAC: c.HuellaSolicitudHMAC,
		vecports.AtributoAlmacenEfectoRef:           c.ContenidoAdmitido.Referencia,
		vecports.AtributoAlmacenObjetoRef:           objeto.Referencia,
		vecports.AtributoAlmacenObjetoVersion:       objeto.Version,
		"persona_ref":                               c.PrincipalID, "hecho_ref": c.RecursoRef,
		"huella_carga_sha256":     datos.HuellaCargaSHA256,
		"huella_contenido_sha256": c.ContenidoAdmitido.HuellaSHA256,
		"tamano":                  strconv.FormatInt(c.ContenidoAdmitido.Tamano, 10),
		"mime":                    c.ContenidoAdmitido.MIME,
		"evidencia_analisis_ref":  c.Analisis.EvidenciaRef,
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: c.ID, ModuloID: "documentos", Tipo: "carga_documental",
		Ambitos: map[string]string{"persona_ref": c.PrincipalID, "hecho_ref": c.RecursoRef,
			"plan_ref": entrada.PlanRef}, Atributos: atributos}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: entrada.VinculoV2, ReferenciaMotivo: s.motivo,
		Accion: vecports.AccionNegocioLeerCargaAdmitidaCustodia, Recurso: recurso,
		Finalidad: entrada.Finalidad, Correlacion: correlacion})
	if err != nil {
		return ports.LecturaCargaAdmitidaCustodia{}, vecdomain.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.emisorAlmacen.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, entrada.ActorRegistradoV2)
	if err != nil || dependenciaNula(exportador) ||
		decision.ExigirProyeccionPara(solicitud, []string{"contenido", "carga_admitida"}, nil) != nil ||
		confirmacion.Validar() != nil {
		return ports.LecturaCargaAdmitidaCustodia{}, vecdomain.ErrAutorizacionDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion,
		entrada.ActorRegistradoV2, s.motivo, material,
		vecports.AudienciaAlmacenLeerCargaAdmitidaCustodiaV3) {
		return ports.LecturaCargaAdmitidaCustodia{}, vecdomain.ErrAutorizacionDenegada
	}
	ahora := s.reloj.Ahora().UTC()
	contexto, err := vecports.NuevoContextoLeerCargaAdmitidaCustodiaAlmacenV3(solicitud, decision,
		confirmacion, vinculos, ahora)
	if err != nil || contexto.ValidarParaEn(vecports.AccionAlmacenLeer, ahora) != nil {
		return ports.LecturaCargaAdmitidaCustodia{}, vecdomain.ErrAutorizacionDenegada
	}
	orden := vecports.SolicitudAbrirObjeto{Contexto: contexto, Objeto: objeto,
		Zona: vecports.ZonaAlmacenAdmitida, Limite: c.ContenidoAdmitido.Tamano}
	lectura, err := s.almacen.Abrir(ctx, orden)
	if err != nil {
		return ports.LecturaCargaAdmitidaCustodia{}, err
	}
	if lectura.ValidarContra(orden) != nil || lectura.Objeto.Objeto != objeto ||
		lectura.Objeto.Zona != vecports.ZonaAlmacenAdmitida ||
		lectura.Objeto.Tamano != c.ContenidoAdmitido.Tamano ||
		lectura.Objeto.HuellaSHA256 != c.ContenidoAdmitido.HuellaSHA256 ||
		lectura.Objeto.MIME != c.ContenidoAdmitido.MIME ||
		lectura.Objeto.EvidenciaCreacionRef != c.ContenidoAdmitido.EvidenciaRef {
		if !dependenciaNula(lectura.Contenido) {
			_ = lectura.Contenido.Close()
		}
		return ports.LecturaCargaAdmitidaCustodia{}, ErrFuenteCargaCustodiaNoDisponible
	}
	return ports.LecturaCargaAdmitidaCustodia{Datos: datos,
		Contenido: &lectorCustodiaCompleta{origen: lectura.Contenido, esperado: c.ContenidoAdmitido.Tamano,
			huella: c.ContenidoAdmitido.HuellaSHA256, hash: sha256.New()}}, nil
}

type lectorCustodiaCompleta struct {
	origen   io.ReadCloser
	esperado int64
	huella   string
	hash     hash.Hash
	leidos   int64
	completo bool
	fallo    error
}

func (l *lectorCustodiaCompleta) Read(p []byte) (int, error) {
	if l.fallo != nil {
		return 0, l.fallo
	}
	if l.completo {
		return 0, io.EOF
	}
	n, err := l.origen.Read(p)
	if n > 0 {
		l.leidos += int64(n)
		_, _ = l.hash.Write(p[:n])
	}
	if l.leidos > l.esperado || (err == io.EOF &&
		(l.leidos != l.esperado || subtle.ConstantTimeCompare([]byte(hex.EncodeToString(l.hash.Sum(nil))), []byte(l.huella)) != 1)) {
		l.fallo = vecports.ErrIntegridadObjetoAlmacen
		return n, l.fallo
	}
	if err == io.EOF {
		l.completo = true
	}
	if err != nil && err != io.EOF {
		l.fallo = err
	}
	return n, err
}

func (l *lectorCustodiaCompleta) Close() error {
	err := l.origen.Close()
	if err != nil {
		return err
	}
	if !l.completo || l.fallo != nil {
		return vecports.ErrIntegridadObjetoAlmacen
	}
	return nil
}

var _ ports.FuenteCargaAdmitidaCustodia = (*ServicioFuenteCargaAdmitidaCustodia)(nil)
