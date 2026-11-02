---
title: Menú, submenús y botón
description: Elige los elementos del menú superior, agrúpalos en submenús y destaca un botón.
weight: 2
---
## Menú automático

Si no eliges un menú, el menú superior muestra las secciones del sitio, en el orden del **weight** de cada una.

## Desde Blup Studio

En la pantalla **Sitio**, bloque **Menú superior**:

1. **Agregar página** elige una carpeta o nota del sitio.
2. **Agregar enlace** crea un elemento que lleva a otra dirección: un nombre y el `https://`.
3. El nombre de cada elemento es libre. Haz clic para editarlo.
4. **Arrastra por el asa** para cambiar el orden.

### Submenús

- **Arrastra un elemento encima de otro**: entra en el submenú del de arriba. El elemento de arriba pasa a solo abrir el submenú, sin enlace.
- **Nuevo submenú** crea una entrada vacía con un nombre. Arrastra elementos dentro de ella.
- Los botones de cada elemento hacen lo mismo sin mouse: subir, bajar, **Al submenú de arriba**, **Sacar del submenú** y quitar.
- Hay un solo nivel: un submenú no entra en otro.

### Botón superior

En el bloque **Botón superior**, escribe el **Texto del botón** y **A dónde lleva**. Aparece a la derecha del menú, destacado. Deja el texto vacío para no tener botón.

## Desde Obsidian

En la nota **_index** de la carpeta del sitio, un elemento por línea:

```yaml
menu:
  - Documentación
  - Ayuda | Centro de ayuda
  - Recursos |
  - + Novedades | Changelog
  - + Roadmap
  - Precios | https://acme.com/precios
button_text: Crear cuenta
button_link: https://app.acme.com/registro
```

| Línea | Se convierte en |
| --- | --- |
| `Documentación` | Enlace a la carpeta (o nota) con ese nombre o título. |
| `Ayuda \| Centro de ayuda` | El texto "Ayuda", que lleva a la carpeta "Centro de ayuda". |
| `Recursos \|` | Una entrada que solo abre el submenú. |
| `+ Novedades \| Changelog` | Con `+` delante, el elemento entra en el submenú de la línea de arriba. |
| `Precios \| https://...` | Enlace a otro sitio. |
| `Inicio` | La página de inicio. |
| `Blog \| /blog/` | Una dirección del propio sitio. |

El elemento de la página actual queda marcado.

## Desde la línea de comandos

`blup customize` (o `blup personalizar`) > menú arma el menú eligiendo carpetas, etiquetas y enlaces, y pregunta por el botón. Los submenús no se rehacen desde la terminal: te avisa y te deja editarlos en el Studio o en Obsidian.
