package httpcopias

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

type controlRevisionTest struct {
	propuesta p.Propuesta
	efectos   int
}

func (c *controlRevisionTest) Proponer(context.Context, p.Sesion, p.SolicitudPropuesta) (p.Propuesta, error) {
	return p.Propuesta{}, p.ErrNoDisponible
}
func (c *controlRevisionTest) Revisar(context.Context, p.Sesion, string, p.SolicitudControl) (p.Propuesta, error) {
	c.efectos++
	return c.propuesta, nil
}
func (c *controlRevisionTest) Ejecutar(context.Context, p.Sesion, string, p.SolicitudControl) (p.Recibo, error) {
	return p.Recibo{}, p.ErrNoDisponible
}

type fuenteRevisionTest struct {
	propuesta p.Propuesta
	err       error
	llamadas  int
}

func (f *fuenteRevisionTest) PropuestaParaRevision(context.Context, p.Sesion, string) (p.Propuesta, error) {
	f.llamadas++
	return f.propuesta, f.err
}

type consultasRevisionTest struct {
	backendTest
	propuesta p.Propuesta
}

func (c *consultasRevisionTest) Propuestas(context.Context, p.Sesion) ([]p.Propuesta, error) {
	return []p.Propuesta{c.propuesta}, nil
}

func propuestaInformada() p.Propuesta {
	now := time.Now().UTC().Truncate(time.Microsecond)
	hash := strings.Repeat("a", 64)
	v := p.Propuesta{PropuestaRef: "propuesta_aaaaaaaa", ConjuntoRef: "conjunto_aaaaaaaa", DestinoRef: "destino_aaaaaaaa", Estado: "pendiente", Version: 5, HuellaSHA256: hash, ConjuntoHuellaSHA256: hash, PreimagenSHA256: hash, PoliticaRef: "politica_aaaaaaaa", PoliticaHuellaSHA256: hash, MotivoRef: "motivo_aaaaaaaa", VentanaRef: "ventana_aaaaaaaa", DobleControl: true, CopiaPreviaRequerida: true, CaducaEn: now.Add(time.Hour), VentanaInicio: now.Add(-time.Hour), VentanaFin: now.Add(2 * time.Hour)}
	version := p.VersionObservada{ReleaseRef: "release_aaaaaaaa", AppVersion: "1.0.0", PostgreSQLVersion: "18.4", EsquemaRef: "esquema_aaaaaaaa", DescriptorHuellaSHA256: hash}
	v.MetadatosRevision = &p.MetadatosRevision{PropuestaRef: v.PropuestaRef, PropuestaHuellaSHA256: hash, ConjuntoRef: v.ConjuntoRef, ConjuntoHuellaSHA256: hash, DestinoRef: v.DestinoRef, PreimagenSHA256: hash, FechaCopia: now.Add(-24 * time.Hour), PerdidaDesde: now.Add(-24 * time.Hour), Actual: version, Resultante: version, Compatibilidad: p.CompatibilidadRevision{Estado: "compatible", Razones: []string{"api.admin.copias.compatibilidad.comprobacion_conjunto_compatible"}}, ObservadaEn: now}
	return v
}
func revisionBody(v p.Propuesta) string {
	data, _ := json.Marshal(p.SolicitudControl{OperacionRef: "operacion_aaaaaaaa", DestinoRef: v.DestinoRef, PropuestaHuellaSHA256: v.HuellaSHA256, VersionEsperada: v.Version})
	return string(data)
}

func TestRevisionExigeMetadataCanonicaAntesDelEfecto(t *testing.T) {
	for _, tc := range []struct {
		name      string
		modify    func(*p.Propuesta)
		sinFuente bool
		status    int
	}{
		{"sin fuente", func(*p.Propuesta) {}, true, 503},
		{"politica ausente", func(v *p.Propuesta) { v.PoliticaRef = "" }, false, 503},
		{"politica huella ausente", func(v *p.Propuesta) { v.PoliticaHuellaSHA256 = "" }, false, 503},
		{"motivo ausente", func(v *p.Propuesta) { v.MotivoRef = "" }, false, 503},
		{"referencia ventana ausente", func(v *p.Propuesta) { v.VentanaRef = "" }, false, 503},
		{"razon desconocida", func(v *p.Propuesta) {
			v.MetadatosRevision.Compatibilidad.Razones = []string{"api.admin.copias.compatibilidad.desconocida"}
		}, false, 503},
		{"razon prefijo ajeno", func(v *p.Propuesta) {
			v.MetadatosRevision.Compatibilidad.Razones = []string{"api.admin.otro.compatible"}
		}, false, 503},
		{"razon fuera compatibilidad", func(v *p.Propuesta) {
			v.MetadatosRevision.Compatibilidad.Razones = []string{"api.admin.copias.error.acceso_denegado"}
		}, false, 503},
		{"razon negativa incompatible", func(v *p.Propuesta) {
			v.MetadatosRevision.Compatibilidad.Razones = []string{"api.admin.copias.compatibilidad.incompatible"}
		}, false, 503},
		{"razon negativa no comprobable", func(v *p.Propuesta) {
			v.MetadatosRevision.Compatibilidad.Razones = []string{"api.admin.copias.compatibilidad.no_comprobable"}
		}, false, 503},
		{"historia sin metadata", func(v *p.Propuesta) { v.MetadatosRevision = nil }, false, 503},
		{"fecha copia ausente", func(v *p.Propuesta) { v.MetadatosRevision.FechaCopia = time.Time{} }, false, 503},
		{"perdida ausente", func(v *p.Propuesta) { v.MetadatosRevision.PerdidaDesde = time.Time{} }, false, 503},
		{"PG desconocido", func(v *p.Propuesta) { v.MetadatosRevision.Actual.PostgreSQLVersion = "" }, false, 503},
		{"app final desconocida", func(v *p.Propuesta) { v.MetadatosRevision.Resultante.AppVersion = "" }, false, 503},
		{"schema desconocido", func(v *p.Propuesta) { v.MetadatosRevision.Resultante.EsquemaRef = "" }, false, 503},
		{"descriptor sin huella", func(v *p.Propuesta) { v.MetadatosRevision.Actual.DescriptorHuellaSHA256 = "" }, false, 503},
		{"destino distinto metadata", func(v *p.Propuesta) { v.MetadatosRevision.DestinoRef = "destino_bbbbbbbb" }, false, 503},
		{"conjunto hash distinto", func(v *p.Propuesta) { v.MetadatosRevision.ConjuntoHuellaSHA256 = strings.Repeat("b", 64) }, false, 503},
		{"propuesta hash distinto", func(v *p.Propuesta) { v.MetadatosRevision.PropuestaHuellaSHA256 = strings.Repeat("b", 64) }, false, 503},
		{"preimagen distinta", func(v *p.Propuesta) { v.MetadatosRevision.PreimagenSHA256 = strings.Repeat("b", 64) }, false, 503},
		{"observacion futura", func(v *p.Propuesta) { v.MetadatosRevision.ObservadaEn = time.Now().UTC().Add(time.Hour) }, false, 503},
		{"compatibilidad desconocida", func(v *p.Propuesta) { v.MetadatosRevision.Compatibilidad.Estado = "desconocida" }, false, 503},
		{"razones ausentes", func(v *p.Propuesta) { v.MetadatosRevision.Compatibilidad.Razones = nil }, false, 503},
		{"ventana ausente", func(v *p.Propuesta) { v.VentanaInicio = time.Time{} }, false, 503},
		{"CAS distinto fuente", func(v *p.Propuesta) { v.Version++ }, false, 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := propuestaInformada()
			body := revisionBody(v)
			tc.modify(&v)
			h, _, _ := preparar(t)
			c := &controlRevisionTest{propuesta: v}
			h.servicio.Control = c
			if !tc.sinFuente {
				h.servicio.FuenteRevision = &fuenteRevisionTest{propuesta: v}
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion("POST", PrefijoV1+"/propuestas/"+v.PropuestaRef+"/revision", body))
			if w.Code != tc.status || c.efectos != 0 {
				t.Fatalf("status=%d effects=%d", w.Code, c.efectos)
			}
			if tc.status == 503 {
				h.servicio.Lecturas = &consultasRevisionTest{propuesta: v}
				capWriter := httptest.NewRecorder()
				h.ServeHTTP(capWriter, peticion("GET", PrefijoV1+"/capacidades", ""))
				var result struct {
					Capacidades p.Capacidades `json:"capacidades"`
				}
				if capWriter.Code != 200 || json.Unmarshal(capWriter.Body.Bytes(), &result) != nil || result.Capacidades.Revisar {
					t.Fatal("incomplete or unknown canonical source still enabled review")
				}
			}

		})
	}
}

func TestRevisionLeeSourceActualYRechazaMetadataCliente(t *testing.T) {
	v := propuestaInformada()
	h, _, _ := preparar(t)
	source := &fuenteRevisionTest{propuesta: v}
	c := &controlRevisionTest{propuesta: v}
	h.servicio.Control = c
	h.servicio.FuenteRevision = source
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion("POST", PrefijoV1+"/propuestas/"+v.PropuestaRef+"/revision", revisionBody(v)))
	if w.Code != 200 || source.llamadas != 1 || c.efectos != 1 {
		t.Fatalf("status=%d source=%d effects=%d", w.Code, source.llamadas, c.efectos)
	}
	source.propuesta.MetadatosRevision = nil
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticion("POST", PrefijoV1+"/propuestas/"+v.PropuestaRef+"/revision", revisionBody(v)))
	if w.Code != 503 || source.llamadas != 2 || c.efectos != 1 {
		t.Fatal("source not rechecked")
	}
	for _, key := range []string{"metadatos_revision", "Metadatos_revision"} {
		body := revisionBody(v)
		body = body[:len(body)-1] + `,"` + key + `":{}}`
		w = httptest.NewRecorder()
		h.ServeHTTP(w, peticion("POST", PrefijoV1+"/propuestas/"+v.PropuestaRef+"/revision", body))
		if w.Code != 400 || source.llamadas != 2 || c.efectos != 1 {
			t.Fatal("client metadata reached source/effect")
		}
	}
}

func TestCapacidadRevisionNoSurgeDelPermisoSolo(t *testing.T) {
	h, _, _ := preparar(t)
	v := propuestaInformada()
	h.servicio.Control = &controlRevisionTest{propuesta: v}
	h.servicio.Lecturas = &consultasRevisionTest{propuesta: v}
	read := func() bool {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion("GET", PrefijoV1+"/capacidades", ""))
		if w.Code != 200 {
			t.Fatalf("capabilities status=%d", w.Code)
		}
		var result struct {
			Capacidades p.Capacidades `json:"capacidades"`
		}
		if json.Unmarshal(w.Body.Bytes(), &result) != nil {
			t.Fatal("invalid JSON")
		}
		return result.Capacidades.Revisar
	}
	if read() {
		t.Fatal("permission and control implicitly enabled review")
	}
	source := &fuenteRevisionTest{propuesta: v}
	h.servicio.FuenteRevision = source
	if !read() {
		t.Fatal("canonical metadata did not enable review")
	}
	source.propuesta.MetadatosRevision = nil
	if read() {
		t.Fatal("missing metadata still enabled review")
	}
}

func TestPropuestaExponeMetadataCanonicaYConservaHistoriaIncompleta(t *testing.T) {
	for _, complete := range []bool{true, false} {
		v := propuestaInformada()
		if !complete {
			v.MetadatosRevision = nil
		}
		h, _, _ := preparar(t)
		h.servicio.Lecturas = &consultasRevisionTest{propuesta: v}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion("GET", PrefijoV1+"/propuestas", ""))
		if w.Code != 200 {
			t.Fatalf("status=%d", w.Code)
		}
		var dto struct {
			Propuestas []p.Propuesta `json:"propuestas"`
		}
		if json.Unmarshal(w.Body.Bytes(), &dto) != nil || len(dto.Propuestas) != 1 {
			t.Fatal("proposal output shape")
		}
		got := dto.Propuestas[0]
		if got.PropuestaRef != v.PropuestaRef || got.HuellaSHA256 != v.HuellaSHA256 {
			t.Fatal("canonical proposal bindings lost")
		}
		if !complete {
			if got.MetadatosRevision != nil {
				t.Fatal("missing metadata replaced by defaults")
			}
			continue
		}
		if got.MetadatosRevision == nil || !got.MetadatosRevision.FechaCopia.Equal(v.MetadatosRevision.FechaCopia) || got.MetadatosRevision.Resultante != v.MetadatosRevision.Resultante {
			t.Fatal("observed metadata changed in transport")
		}
		for _, field := range []string{`"metadatos_revision"`, `"fecha_copia"`, `"perdida_desde"`, `"actual"`, `"resultante"`, `"postgresql_version"`, `"descriptor_huella_sha256"`, `"compatibilidad"`, `"razones"`, `"observada_en"`} {
			if !strings.Contains(w.Body.String(), field) {
				t.Fatalf("missing UI schema field %s", field)
			}
		}
	}
}

func TestRevisionAdmiteSoloRazonesPositivasConocidas(t *testing.T) {
	for _, key := range []string{"api.admin.copias.compatibilidad.compatible", "api.admin.copias.compatibilidad.comprobacion_conjunto_compatible"} {
		v := propuestaInformada()
		v.MetadatosRevision.Compatibilidad.Razones = []string{key}
		h, _, _ := preparar(t)
		c := &controlRevisionTest{propuesta: v}
		h.servicio.Control = c
		h.servicio.FuenteRevision = &fuenteRevisionTest{propuesta: v}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion("POST", PrefijoV1+"/propuestas/"+v.PropuestaRef+"/revision", revisionBody(v)))
		if w.Code != 200 || c.efectos != 1 {
			t.Fatalf("known positive %s rejected: status=%d effects=%d", key, w.Code, c.efectos)
		}
	}
}
