package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"vec-diputacion-granada/config"
)

var (
	ErrActivacionDesarrolloInvalida    = errors.New("bootstrap: activacion de desarrollo incompleta o incompatible")
	ErrProveedorDesarrolloEnProduccion = errors.New("bootstrap: proveedor de desarrollo compuesto en produccion")
	ErrComposicionDesarrolloIncompleta = errors.New("bootstrap: composicion de proveedores de desarrollo incompleta")
	ErrRegistroArranqueDesarrollo      = errors.New("bootstrap: no se pudo registrar el arranque no autoritativo")
)

const (
	EsquemaMetadatosNoAutoritativos = "vec.acto.no-autoritativo.v1"
	AutoridadNoAutoritativa         = "no_autoritativo"
	proveedorSeguridadDesarrolloRef = "proveedor:seguridad:desarrollo:t21"
)

type TipoProveedorSeguridad string

const (
	ProveedorIdentidad    TipoProveedorSeguridad = "identidad"
	ProveedorIdempotencia TipoProveedorSeguridad = "idempotencia_hmac"
	ProveedorKMS          TipoProveedorSeguridad = "kms"
	ProveedorTSA          TipoProveedorSeguridad = "tsa"
	ProveedorTLS          TipoProveedorSeguridad = "tls"
)

var tiposProveedorDesarrollo = []TipoProveedorSeguridad{
	ProveedorIdentidad,
	ProveedorIdempotencia,
	ProveedorKMS,
	ProveedorTSA,
	ProveedorTLS,
}

type procedenciaProveedor string

const (
	procedenciaAutoritativa procedenciaProveedor = "autoritativo"
	procedenciaDesarrollo   procedenciaProveedor = "desarrollo"
)

// DescriptorProveedorSeguridad clasifica un adaptador ya inyectado en la raiz
// de composicion. No sustituye a ninguna interfaz KMS, TSA, TLS o de identidad:
// es solo la prueba anti-fuga que acompana al adaptador concreto.
type DescriptorProveedorSeguridad struct {
	tipo        TipoProveedorSeguridad
	referencia  string
	procedencia procedenciaProveedor
}

func NuevoDescriptorProveedorDesarrollo(tipo TipoProveedorSeguridad, referencia string) (DescriptorProveedorSeguridad, error) {
	return nuevoDescriptorProveedor(tipo, referencia, procedenciaDesarrollo)
}

func NuevoDescriptorProveedorAutoritativo(tipo TipoProveedorSeguridad, referencia string) (DescriptorProveedorSeguridad, error) {
	return nuevoDescriptorProveedor(tipo, referencia, procedenciaAutoritativa)
}

func nuevoDescriptorProveedor(
	tipo TipoProveedorSeguridad,
	referencia string,
	procedencia procedenciaProveedor,
) (DescriptorProveedorSeguridad, error) {
	referencia = strings.TrimSpace(referencia)
	if !esTipoProveedorConocido(tipo) || !esReferenciaOpacaValida(referencia) {
		return DescriptorProveedorSeguridad{}, fmt.Errorf("%w: descriptor de proveedor no valido", ErrComposicionDesarrolloIncompleta)
	}
	return DescriptorProveedorSeguridad{tipo: tipo, referencia: referencia, procedencia: procedencia}, nil
}

// ReferenciaProveedorNoAutoritativo es la representacion persistible y sin
// secretos de un proveedor local. La referencia es opaca: nunca contiene una
// ruta, un certificado, una clave ni su contenido.
type ReferenciaProveedorNoAutoritativo struct {
	Tipo       TipoProveedorSeguridad `json:"tipo"`
	Referencia string                 `json:"referencia"`
}

// DatosMetadatosNoAutoritativos describe la composicion y alimenta el aviso de
// arranque. No es el contrato durable de un acto: ese contrato canonico es
// gobiernoconvocatorias.ProcedenciaActoBorrador.
type DatosMetadatosNoAutoritativos struct {
	Esquema                    string                              `json:"esquema"`
	Autoridad                  string                              `json:"autoridad"`
	PerfilEjecucion            string                              `json:"perfil_ejecucion"`
	MigrableAProduccion        bool                                `json:"migrable_a_produccion"`
	DescartableAlCambiarPerfil bool                                `json:"descartable_al_cambiar_perfil"`
	Proveedores                []ReferenciaProveedorNoAutoritativo `json:"proveedores"`
}

// MetadatosNoAutoritativos conserva inmutable el inventario de proveedores de
// la composicion. Datos devuelve siempre copia y nunca sustituye la procedencia
// que el nucleo obliga a persistir con cada acto.
type MetadatosNoAutoritativos struct {
	datos DatosMetadatosNoAutoritativos
}

func (m MetadatosNoAutoritativos) Datos() DatosMetadatosNoAutoritativos {
	datos := m.datos
	datos.Proveedores = append([]ReferenciaProveedorNoAutoritativo(nil), m.datos.Proveedores...)
	return datos
}

func (m MetadatosNoAutoritativos) MarshalJSON() ([]byte, error) {
	if err := validarMetadatosNoAutoritativos(m.datos); err != nil {
		return nil, err
	}
	return json.Marshal(m.Datos())
}

// PrepararPerfilEjecucion es la unica entrada para habilitar una composicion
// local. Devuelve metadatos de composicion y emite el aviso de arranque antes
// de permitir continuar. En produccion devuelve el valor cero.
func PrepararPerfilEjecucion(
	cfg config.Config,
	proveedores []DescriptorProveedorSeguridad,
	registro io.Writer,
) (MetadatosNoAutoritativos, error) {
	cfg = cfg.Normalize()
	metadatos, desarrollo, err := validarComposicionPerfil(cfg, proveedores)
	if err != nil {
		return MetadatosNoAutoritativos{}, err
	}
	if !desarrollo {
		return MetadatosNoAutoritativos{}, nil
	}
	if registro == nil {
		return MetadatosNoAutoritativos{}, ErrRegistroArranqueDesarrollo
	}
	aviso := struct {
		Nivel       string                   `json:"nivel"`
		Evento      string                   `json:"evento"`
		Perfil      string                   `json:"perfil"`
		Autoridad   string                   `json:"autoridad"`
		Proveedores []TipoProveedorSeguridad `json:"proveedores"`
	}{
		Nivel:       "ADVERTENCIA",
		Evento:      "arranque_con_credenciales_no_autoritativas",
		Perfil:      config.ExecutionProfileDevelopment,
		Autoridad:   AutoridadNoAutoritativa,
		Proveedores: append([]TipoProveedorSeguridad(nil), tiposProveedorDesarrollo...),
	}
	if err := json.NewEncoder(registro).Encode(aviso); err != nil {
		return MetadatosNoAutoritativos{}, errors.Join(ErrRegistroArranqueDesarrollo, err)
	}
	return metadatos, nil
}

func validarComposicionPerfil(
	cfg config.Config,
	proveedores []DescriptorProveedorSeguridad,
) (MetadatosNoAutoritativos, bool, error) {
	hayProveedorDesarrollo := false
	for _, proveedor := range proveedores {
		if proveedor.procedencia == procedenciaDesarrollo {
			hayProveedorDesarrollo = true
		}
	}

	if cfg.ExecutionProfile != config.ExecutionProfileDevelopment {
		if hayProveedorDesarrollo {
			return MetadatosNoAutoritativos{}, false, ErrProveedorDesarrolloEnProduccion
		}
		if cfg.AuthMode == config.AuthModeDevelopment || cfg.DevelopmentGuard != "" || cfg.DevelopmentMaterialDir != "" {
			return MetadatosNoAutoritativos{}, false, ErrActivacionDesarrolloInvalida
		}
		return MetadatosNoAutoritativos{}, false, nil
	}

	if !cfg.DevelopmentEnabledByDoubleKey() || cfg.DevelopmentMaterialDir == "" || !filepath.IsAbs(cfg.DevelopmentMaterialDir) {
		return MetadatosNoAutoritativos{}, false, ErrActivacionDesarrolloInvalida
	}

	referencias := make(map[TipoProveedorSeguridad]string, len(tiposProveedorDesarrollo))
	for _, proveedor := range proveedores {
		if proveedor.procedencia != procedenciaDesarrollo || !esTipoProveedorConocido(proveedor.tipo) ||
			!esReferenciaOpacaValida(proveedor.referencia) {
			return MetadatosNoAutoritativos{}, false, ErrComposicionDesarrolloIncompleta
		}
		if _, duplicado := referencias[proveedor.tipo]; duplicado {
			return MetadatosNoAutoritativos{}, false, ErrComposicionDesarrolloIncompleta
		}
		referencias[proveedor.tipo] = proveedor.referencia
	}
	if len(referencias) != len(tiposProveedorDesarrollo) {
		return MetadatosNoAutoritativos{}, false, ErrComposicionDesarrolloIncompleta
	}

	ordenados := append([]TipoProveedorSeguridad(nil), tiposProveedorDesarrollo...)
	sort.Slice(ordenados, func(i, j int) bool { return ordenados[i] < ordenados[j] })
	proveedoresPersistibles := make([]ReferenciaProveedorNoAutoritativo, 0, len(ordenados))
	for _, tipo := range ordenados {
		proveedoresPersistibles = append(proveedoresPersistibles, ReferenciaProveedorNoAutoritativo{
			Tipo:       tipo,
			Referencia: referencias[tipo],
		})
	}
	metadatos := MetadatosNoAutoritativos{datos: DatosMetadatosNoAutoritativos{
		Esquema:                    EsquemaMetadatosNoAutoritativos,
		Autoridad:                  AutoridadNoAutoritativa,
		PerfilEjecucion:            config.ExecutionProfileDevelopment,
		MigrableAProduccion:        false,
		DescartableAlCambiarPerfil: true,
		Proveedores:                proveedoresPersistibles,
	}}
	if err := validarMetadatosNoAutoritativos(metadatos.datos); err != nil {
		return MetadatosNoAutoritativos{}, false, err
	}
	return metadatos, true, nil
}

func validarMetadatosNoAutoritativos(datos DatosMetadatosNoAutoritativos) error {
	if datos.Esquema != EsquemaMetadatosNoAutoritativos || datos.Autoridad != AutoridadNoAutoritativa ||
		datos.PerfilEjecucion != config.ExecutionProfileDevelopment || datos.MigrableAProduccion ||
		!datos.DescartableAlCambiarPerfil || len(datos.Proveedores) != len(tiposProveedorDesarrollo) {
		return ErrComposicionDesarrolloIncompleta
	}
	for _, proveedor := range datos.Proveedores {
		if !esTipoProveedorConocido(proveedor.Tipo) || !esReferenciaOpacaValida(proveedor.Referencia) {
			return ErrComposicionDesarrolloIncompleta
		}
	}
	return nil
}

func esTipoProveedorConocido(tipo TipoProveedorSeguridad) bool {
	for _, conocido := range tiposProveedorDesarrollo {
		if tipo == conocido {
			return true
		}
	}
	return false
}

func esReferenciaOpacaValida(referencia string) bool {
	if len(referencia) < 1 || len(referencia) > 128 || strings.ContainsAny(referencia, "/\\ \t\r\n") {
		return false
	}
	for _, caracter := range referencia {
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= '0' && caracter <= '9') ||
			caracter == '-' || caracter == '_' || caracter == '.' || caracter == ':' {
			continue
		}
		return false
	}
	return true
}
