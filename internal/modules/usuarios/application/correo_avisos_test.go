package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorAvisosPrueba struct {
	err       error
	audiencia string
	operacion string
	efecto    string
	huella    string
	llamadas  int
	material  []byte
}

func (p *proveedorAvisosPrueba) ProveerMaterialCorreoAvisos(_ context.Context, m ports.MaterialCorreoAvisos, material []byte) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	p.material = append([]byte(nil), material...)
	if p.err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, p.err
	}
	recurso, _, err := canonico.RecursoCorreoAvisos(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	huella, _ := recurso.HuellaContextoAutorizacionSHA256()
	audiencia, operacion, efecto := ports.AudienciaCorreoAvisosLlamamientoInterna, ports.AccionV3CorreoAvisosLlamamiento, m.BolsaRef
	if p.audiencia != "" {
		audiencia = p.audiencia
	}
	if p.operacion != "" {
		operacion = p.operacion
	}
	if p.efecto != "" {
		efecto = p.efecto
	}
	if p.huella != "" {
		huella = p.huella
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_avisos", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), operacion, efecto, huella, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

type registroAvisosPrueba struct {
	lectura  ports.LecturaCorreoAvisos
	err      error
	llamadas int
	material []byte
}

func (r *registroAvisosPrueba) LeerCorreoActivoAvisos(_ context.Context, material []byte, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.LecturaCorreoAvisos, error) {
	r.llamadas++
	r.material = append([]byte(nil), material...)
	return r.lectura, r.err
}

type protectorAvisosPrueba struct {
	claro   string
	err     error
	persona string
}

func (p *protectorAvisosPrueba) CifrarDireccionCorreo(context.Context, string, string, uint64, []byte) (ports.SobreDireccionCorreo, error) {
	return ports.SobreDireccionCorreo{}, errors.New("no se usa")
}

func (p *protectorAvisosPrueba) ConDireccionCorreoDescifrada(_ context.Context, persona string, _ ports.SobreDireccionCorreo, usar func([]byte) error) error {
	p.persona = persona
	if p.err != nil {
		return p.err
	}
	return usar([]byte(p.claro))
}

func solicitudAvisosAppPrueba() ports.SolicitudCorreoAvisos {
	return ports.SolicitudCorreoAvisos{BolsaRef: "bolsa:1", UnidadRef: "unidad:1", AmbitoRef: "ambito:1", LlamamientoRef: "llamamiento:" + strings.Repeat("0f", 32), CandidatoRef: "can_" + strings.Repeat("a", 24)}
}

func lecturaEncontradaPrueba() ports.LecturaCorreoAvisos {
	return ports.LecturaCorreoAvisos{Encontrado: true, PersonaRef: "per_" + strings.Repeat("p", 24), CorreoRef: refCorreoPrueba, AuditoriaRef: "aud_1", Sobre: ports.SobreDireccionCorreo{CorreoRef: refCorreoPrueba, Version: 1}}
}

func TestCorreoAvisosEntregaSoloElActivoDentroDeLaLlamada(t *testing.T) {
	registro := &registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}
	protector := &protectorAvisosPrueba{claro: "activa@personal.example.org"}
	s, err := NuevoServicioCorreoAvisos(registro, protector)
	if err != nil {
		t.Fatal(err)
	}
	proveedor := &proveedorAvisosPrueba{}
	var usadas []string
	r, err := s.ConCorreoActivoAvisos(context.Background(), ports.OrdenCorreoAvisos{Proveedor: proveedor}, solicitudAvisosAppPrueba(), func(d string) { usadas = append(usadas, d) })
	if err != nil || !r.Encontrado || r.CorreoRef != refCorreoPrueba || len(usadas) != 1 || usadas[0] != "activa@personal.example.org" {
		t.Fatalf("%+v %v %v", r, err, usadas)
	}
	if !bytes.Equal(proveedor.material, registro.material) || protector.persona != registro.lectura.PersonaRef {
		t.Fatal("la V3 y la lectura deben usar el mismo material; el sobre se abre con la persona de la lectura")
	}
	if bytes.Contains(registro.material, []byte("per_")) || bytes.Contains(registro.material, []byte("@")) {
		t.Fatalf("el material no lleva datos personales: %s", registro.material)
	}
}

func TestCorreoAvisosSinActivoNoLlamaAUsar(t *testing.T) {
	s, _ := NuevoServicioCorreoAvisos(&registroAvisosPrueba{lectura: ports.LecturaCorreoAvisos{AuditoriaRef: "aud_1"}}, &protectorAvisosPrueba{claro: "x@example.org"})
	llamado := false
	r, err := s.ConCorreoActivoAvisos(context.Background(), ports.OrdenCorreoAvisos{Proveedor: &proveedorAvisosPrueba{}}, solicitudAvisosAppPrueba(), func(string) { llamado = true })
	if err != nil || r.Encontrado || r.CorreoRef != "" || llamado {
		t.Fatalf("%+v %v %v", r, err, llamado)
	}
}

func TestCorreoAvisosFallosNoLlamanAUsar(t *testing.T) {
	sobreAjeno := lecturaEncontradaPrueba()
	sobreAjeno.Sobre.CorreoRef = "correo:ffffffffffffffffffffffffffffffff"
	refMala := lecturaEncontradaPrueba()
	refMala.CorreoRef, refMala.Sobre.CorreoRef = "correo:1", "correo:1"
	casos := []struct {
		nombre    string
		proveedor *proveedorAvisosPrueba
		registro  *registroAvisosPrueba
		protector *protectorAvisosPrueba
		esperado  error
		lecturas  int
	}{
		{"V3 denegada", &proveedorAvisosPrueba{err: ports.ErrCorreosProhibido}, &registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosProhibido, 0},
		{"V3 no disponible", &proveedorAvisosPrueba{err: errors.New("caída")}, &registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosNoDisponible, 0},
		{"otra audiencia", &proveedorAvisosPrueba{audiencia: ports.AudienciaConsultarCorreosInterna}, &registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosProhibido, 0},
		{"otra operación", &proveedorAvisosPrueba{operacion: "vec.correos.consultar"}, &registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosProhibido, 0},
		{"otra bolsa", &proveedorAvisosPrueba{efecto: "bolsa:2"}, &registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosProhibido, 0},
		{"otra huella", &proveedorAvisosPrueba{huella: strings.Repeat("d", 64)}, &registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosProhibido, 0},
		{"SQL deniega", &proveedorAvisosPrueba{}, &registroAvisosPrueba{err: ports.ErrCorreosProhibido}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosProhibido, 1},
		{"SQL caída", &proveedorAvisosPrueba{}, &registroAvisosPrueba{err: errors.New("40001")}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosNoDisponible, 1},
		{"sobre de otro correo", &proveedorAvisosPrueba{}, &registroAvisosPrueba{lectura: sobreAjeno}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosNoDisponible, 1},
		{"referencia mal formada", &proveedorAvisosPrueba{}, &registroAvisosPrueba{lectura: refMala}, &protectorAvisosPrueba{claro: "a@example.org"}, ports.ErrCorreosNoDisponible, 1},
		{"sobre que no abre", &proveedorAvisosPrueba{}, &registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}, &protectorAvisosPrueba{err: errors.New("aead")}, ports.ErrCorreosNoDisponible, 1},
		{"dirección no admitida", &proveedorAvisosPrueba{}, &registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}, &protectorAvisosPrueba{claro: "Nombre <a@example.org>"}, ports.ErrCorreosNoDisponible, 1},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, _ := NuevoServicioCorreoAvisos(c.registro, c.protector)
			llamado := false
			r, err := s.ConCorreoActivoAvisos(context.Background(), ports.OrdenCorreoAvisos{Proveedor: c.proveedor}, solicitudAvisosAppPrueba(), func(string) { llamado = true })
			if !errors.Is(err, c.esperado) || llamado || r.Encontrado || c.registro.llamadas != c.lecturas {
				t.Fatalf("%+v %v llamado=%v lecturas=%d", r, err, llamado, c.registro.llamadas)
			}
		})
	}
}

func TestCorreoAvisosRechazaSolicitudesYDependenciasInvalidas(t *testing.T) {
	if _, err := NuevoServicioCorreoAvisos(nil, &protectorAvisosPrueba{}); err == nil {
		t.Fatal("registro nulo admitido")
	}
	s, _ := NuevoServicioCorreoAvisos(&registroAvisosPrueba{lectura: lecturaEncontradaPrueba()}, &protectorAvisosPrueba{claro: "a@example.org"})
	proveedor := &proveedorAvisosPrueba{}
	mala := solicitudAvisosAppPrueba()
	mala.CandidatoRef = "per_" + strings.Repeat("a", 24)
	if _, err := s.ConCorreoActivoAvisos(context.Background(), ports.OrdenCorreoAvisos{Proveedor: proveedor}, mala, func(string) {}); !errors.Is(err, ports.ErrCorreosInvalidos) || proveedor.llamadas != 0 {
		t.Fatalf("una persona no es un candidato: %v", err)
	}
	if _, err := s.ConCorreoActivoAvisos(context.Background(), ports.OrdenCorreoAvisos{}, solicitudAvisosAppPrueba(), func(string) {}); err == nil {
		t.Fatal("orden sin proveedor admitida")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := s.ConCorreoActivoAvisos(ctx, ports.OrdenCorreoAvisos{Proveedor: proveedor}, solicitudAvisosAppPrueba(), func(string) {}); !errors.Is(err, context.Canceled) || proveedor.llamadas != 0 {
		t.Fatalf("contexto cancelado: %v", err)
	}
}
