package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registradorPerfilBolsaPrueba struct {
	llamadas int
	datos    vecports.DatosOrdenIntentoAuditoria
}

func (r *registradorPerfilBolsaPrueba) AppendIntentoAuditoria(ctx context.Context, orden vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	datos, err := orden.Datos()
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, err
	}
	r.llamadas++
	r.datos = datos
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_perfil", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC()}, nil
}

func TestPerfilBolsaFuenteAusenteAuditaErrorSinEntregarDatos(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, _ := escenarioLecturaRespuestasLlamamiento(t, ahora)
	perfil := actor.Resultado.Contexto.PerfilActivoRef
	d := fronteraComunPrueba("consulta-perfil-bolsa", "GET", rutaBolsasRRHHDesarrollo, false)
	d.PerfilesActivosRef = []string{perfil}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: "GET", ruta: rutaBolsasRRHHDesarrollo, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: d})
	proveedor := proveedorContextoComunConsultasRRHHDesarrolloPrueba{resolver: func(context.Context) (contextoSeguridadComunDesarrollo, error) { return actor, nil }}
	seguridad, err := nuevaSeguridadComunDesarrollo(proveedor, &relojSesionConsultaPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	soporte := &soporteSesionBorradorBolsaDesarrollo{soporteCanal: &soporteAltaContratacionTemporalDesarrollo{contexto: ctports.ContextoAutorizacionAltaV3{Resultado: actor.Resultado, Vinculo: actor.Vinculo}}, unidadRef: "unidad:prueba", ambitoRef: "ambito:prueba"}
	registro := &registradorPerfilBolsaPrueba{}
	m := &manejadorParticipacionBolsaDesarrollo{preparador: &preparadorBorradorLlamamientoDesarrollo{sesion: seguridad, soporte: soporte}, registradorPerfil: registro, procesoPerfil: "vec-rrhh"}
	salida, err := m.ConsultarPerfilBolsaRRHH(ctx)
	if salida != nil || !errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible) || registro.llamadas != 1 || registro.datos.Datos.Resultado != vecdomain.ResultadoIntentoAuditoriaError || registro.datos.ResultadoContexto.Contexto.PersonaRef != actor.Resultado.Contexto.PersonaRef {
		t.Fatalf("fuente ausente: datos=%t error=%v asientos=%d", salida != nil, err, registro.llamadas)
	}
}

func TestPerfilBolsaExigeConcesionPropiaYFuenteCoincideConDescriptor(t *testing.T) {
	b, err := os.ReadFile("../../../deploy/principal/fuentes/bolsa_configuracion_rrhh_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Referencia string                                    `json:"referencia"`
		Huella     string                                    `json:"huella_sha256"`
		Canon      string                                    `json:"entradas_canon"`
		Entradas   []vecdomain.EntradaAccionAdministracionV1 `json:"entradas"`
	}
	if json.Unmarshal(b, &f) != nil || len(f.Entradas) != 1 || f.Entradas[0].Validar() != nil {
		t.Fatal("fuente de configuración inválida")
	}
	h := sha256.Sum256([]byte(f.Canon))
	if hex.EncodeToString(h[:]) != f.Huella || f.Entradas[0].FuenteHuellaSHA256 != f.Huella {
		t.Fatal("huella de fuente distinta")
	}
	c := f.Entradas[0].Concesion
	if c.Accion != "bolsa.configuracion.consultar" || c.TipoRecurso != "perfil_rrhh_bolsa" || !reflect.DeepEqual(c.CamposPermitidos, camposConsultaPerfilBolsaRRHH()) || !reflect.DeepEqual(c.Finalidades, []string{"consulta_configuracion_bolsa_rrhh"}) {
		t.Fatal("fuente distinta del descriptor consumido")
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
	recurso := vecdomain.RecursoAutorizable{Referencia: "bolsa:perfil:" + huellaAltaContratacionTemporalDesarrollo(actor.Resultado.Contexto.PerfilActivoRef), ModuloID: "bolsa", Tipo: c.TipoRecurso, Ambitos: map[string]string{"unidad_ref": "unidad:prueba", "ambito_ref": "ambito:prueba"}, Atributos: map[string]string{"perfil_ref": actor.Resultado.Contexto.PerfilActivoRef}}
	negado, err := vecdomain.EvaluarLecturaActualAutorizacionV3(actor.Vinculo, actor.Resultado, fuente.i, c.Accion, recurso, c.Finalidades[0], c.CamposPermitidos, ahora)
	if err == nil && negado.Permitido {
		t.Fatal("la concesión de otro recurso abrió el perfil")
	}
	fuente.i.VersionRol.Concesiones = append(fuente.i.VersionRol.Concesiones, c)
	concedido, err := vecdomain.EvaluarLecturaActualAutorizacionV3(actor.Vinculo, actor.Resultado, fuente.i, c.Accion, recurso, c.Finalidades[0], c.CamposPermitidos, ahora)
	if err != nil || !concedido.Permitido || !reflect.DeepEqual(concedido.Obligaciones, []string{"auditar"}) {
		t.Fatalf("concesión exacta rechazada: %v", err)
	}
}
