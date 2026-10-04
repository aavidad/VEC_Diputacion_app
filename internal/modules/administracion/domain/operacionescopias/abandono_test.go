package operacionescopias

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func abandono(o Operacion) Comando {
	c := comando(o, "abandonar_captura")
	c.Abandono = &ObservacionAbandono{Operacion: o.solicitud.Operacion, Destino: o.solicitud.Destino, FalloReferencia: "fallo:captura", FalloSHA256: strings.Repeat("a", 64), Lease: "lease:captura", EstadoEfecto: "inactivo", EstadoLease: "cancelada", EstadoPlataforma: "sin_efectos_pendientes"}
	if o.Estado() == Capturada || o.Estado() == Verificando {
		c.Abandono.FalloReferencia = "verificacion_fallida"
		c.Abandono.EstadoVerificador = "detenido"
		c.Abandono.EstadoVentana = "inactiva"
	}
	return c
}

func TestAbandonoVerificacionExigeVerificadorDetenidoYVentanaInactiva(t *testing.T) {
	o := verificando(t)
	for _, field := range []string{"verificador_omitido", "verificador_activo", "verificador_incierto", "ventana_omitida", "ventana_activa", "ventana_incierta"} {
		c := abandono(o)
		switch field {
		case "verificador_omitido":
			c.Abandono.EstadoVerificador = ""
		case "verificador_activo":
			c.Abandono.EstadoVerificador = "activo"
		case "verificador_incierto":
			c.Abandono.EstadoVerificador = "incierto"
		case "ventana_omitida":
			c.Abandono.EstadoVentana = ""
		case "ventana_activa":
			c.Abandono.EstadoVentana = "activa"
		case "ventana_incierta":
			c.Abandono.EstadoVentana = "incierta"
		}
		if _, _, _, e := o.Aplicar(c); !errors.Is(e, ErrAbandono) {
			t.Fatal(field, e)
		}
	}
	if o.Estado() != Verificando || o.Version() != 3 {
		t.Fatal("phase changed")
	}
}

func TestCapturaLegacyConservaJSONYFalloPublicadoRequiereObservacionNueva(t *testing.T) {
	o, e := Nueva(solicitud())
	if e != nil {
		t.Fatal(e)
	}
	o = aplicar(t, o, comando(o, "iniciar_captura"))
	c := abandono(o)
	b, e := json.Marshal(c.Abandono)
	if e != nil {
		t.Fatal(e)
	}
	legacy := `{"operacion":"operacion:1","destino":"destino:1","fallo_referencia":"fallo:captura","fallo_sha256":"` + strings.Repeat("a", 64) + `","lease":"lease:captura","estado_efecto":"inactivo","estado_lease":"cancelada","estado_plataforma":"sin_efectos_pendientes"}`
	if string(b) != legacy {
		t.Fatal("legacy serialization changed", string(b))
	}
	c.Abandono.FalloReferencia = "verificacion_fallida"
	if _, _, _, e = o.Aplicar(c); !errors.Is(e, ErrAbandono) {
		t.Fatal("publication bypassed observation", e)
	}
	c.Abandono.EstadoVerificador, c.Abandono.EstadoVentana = "detenido", "inactiva"
	if _, _, _, e = o.Aplicar(c); e != nil {
		t.Fatal("confirmed published failure", e)
	}
}

func TestCapturandoNoIgnoraVerificadorOVentanaActivosDeclarados(t *testing.T) {
	o, e := Nueva(solicitud())
	if e != nil {
		t.Fatal(e)
	}
	o = aplicar(t, o, comando(o, "iniciar_captura"))
	for _, field := range []string{"verificador_activo", "verificador_incierto", "ventana_activa", "ventana_incierta"} {
		c := abandono(o)
		c.Abandono.FalloReferencia = "captura_fallida"
		switch field {
		case "verificador_activo":
			c.Abandono.EstadoVerificador = "activo"
		case "verificador_incierto":
			c.Abandono.EstadoVerificador = "incierto"
		case "ventana_activa":
			c.Abandono.EstadoVentana = "activa"
		case "ventana_incierta":
			c.Abandono.EstadoVentana = "incierta"
		}
		if _, _, _, e := o.Aplicar(c); !errors.Is(e, ErrAbandono) {
			t.Fatal(field, e)
		}
	}
	if o.Estado() != Capturando || o.Version() != 1 {
		t.Fatal("active observation changed state")
	}
}

func TestAbandonoDesdeEstadosActivosSinInventarEnsayo(t *testing.T) {
	initial, e := Nueva(solicitud())
	if e != nil {
		t.Fatal(e)
	}
	capturando := aplicar(t, initial, comando(initial, "iniciar_captura"))
	c := comando(capturando, "confirmar_captura")
	c.ManifiestoSHA256 = strings.Repeat("b", 64)
	capturada := aplicar(t, capturando, c)
	for _, o := range []Operacion{initial, capturando, capturada, verificando(t)} {
		t.Run(string(o.Estado()), func(t *testing.T) {
			c := abandono(o)
			before := o.Historia()
			next, event, replay, e := o.Aplicar(c)
			if e != nil || replay || next.Estado() != AbandonadaDeclarada || next.Version() != o.Version()+1 || event.Comando.Evidencia != nil || event.Comando.ManifiestoSHA256 != "" {
				t.Fatal(e, next.Estado())
			}
			if !reflect.DeepEqual(next.Historia()[:len(before)], before) {
				t.Fatal("previous history changed")
			}
			recovered, e := Reconstruir(solicitud(), next.Historia())
			if e != nil || recovered.Estado() != AbandonadaDeclarada {
				t.Fatal(e)
			}
			_, same, replay, e := recovered.Aplicar(c)
			if e != nil || !replay || !reflect.DeepEqual(same, event) {
				t.Fatal("replay", e)
			}
			h := next.Historia()
			h[len(h)-1].Comando.Abandono.Lease = "lease:alterada"
			if next.Historia()[len(h)-1].Comando.Abandono.Lease != "lease:captura" {
				t.Fatal("mutable history")
			}
			c.Clave = "otra:clave"
			c.VersionEsperada = next.Version()
			if _, _, _, e = next.Aplicar(c); !errors.Is(e, ErrTransicion) {
				t.Fatal("terminal rewritten", e)
			}
		})
	}
}

func TestAbandonoRechazaIncertidumbreLeaseYVinculos(t *testing.T) {
	o := verificando(t)
	for _, state := range []string{"activo", "incierto", "pendiente"} {
		c := abandono(o)
		c.Abandono.EstadoEfecto = state
		if _, _, _, e := o.Aplicar(c); !errors.Is(e, ErrAbandono) {
			t.Fatal(state, e)
		}
	}
	for _, field := range []string{"lease", "plataforma", "operacion", "destino", "solicitud", "version"} {
		c := abandono(o)
		want := ErrAbandono
		switch field {
		case "lease":
			c.Abandono.EstadoLease = "vigente"
		case "plataforma":
			c.Abandono.EstadoPlataforma = "restaurando"
		case "operacion":
			c.Abandono.Operacion = "operacion:otra"
			want = ErrVinculo
		case "destino":
			c.Abandono.Destino = "destino:otro"
			want = ErrVinculo
		case "solicitud":
			c.SolicitudSHA256 = strings.Repeat("d", 64)
			want = ErrVinculo
		case "version":
			c.VersionEsperada--
			want = ErrVersion
		}
		if _, _, _, e := o.Aplicar(c); !errors.Is(e, want) {
			t.Fatal(field, e)
		}
	}
	if o.Estado() != Verificando || o.Version() != 3 {
		t.Fatal("failure mutated operation")
	}
}
