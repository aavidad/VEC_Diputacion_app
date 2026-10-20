package bootstrap

import "errors"

// ErrConstitucionBolsaNoDisponible es la denegación genérica del relleno de
// vínculos históricos de Bolsa. La constitución por línea de órdenes se retiró
// en F1-60 (B145): una bolsa solo se constituye por la carga gobernada de
// CONVOCA, que guarda su ámbito.
var ErrConstitucionBolsaNoDisponible = errors.New("bootstrap: constitucion de bolsa no disponible")
