---
title: Notas y propiedades
description: Crea una nota y se vuelve una página. Las propiedades controlan el resto.
weight: 1
---
Para crear una página, crea una nota en la carpeta correcta. Las **propiedades** (el bloque arriba de la nota) son opcionales y controlan cómo aparece. En Obsidian, escribe `---` en la primera línea o usa el comando **Agregar propiedad**.

## En todas las páginas

| Propiedad | Para qué sirve |
| --- | --- |
| `title` | Título de la página. Sin ella, vale el nombre de la nota. |
| `linkTitle` | Título corto para menús y listas. |
| `description` | Resumen en las listas, en Google y en las redes sociales. |
| `weight` | Orden en el menú lateral y en las listas: el menor va primero. |
| `draft` | `true` deja la nota fuera del sitio (solo aparece en la vista previa). |
| `hidden` | `true` la publica, pero la saca de las listas y le pide a Google que no la muestre. |
| `password` | Protege la página con contraseña. Mira [Contraseñas y páginas ocultas](../Configuración/Contraseñas%20y%20páginas%20ocultas.md). |

## En los posts del blog y en el changelog

| Propiedad | Para qué sirve |
| --- | --- |
| `date` | Fecha. Sin ella, vale `created`. |
| `updated` | Fecha de la última actualización. |
| `tags` | Temas; cada etiqueta tiene su página. |
| `image` | Portada, como `"[[mi-portada.jpg]]"`. |
| `authors` | Quién escribió, como `[Ana, Bruno]`. |

## En el roadmap y en el centro de ayuda

| Propiedad | Para qué sirve |
| --- | --- |
| `status` | Columna del roadmap: planeado, en progreso o completado. |
| `category` | Etiqueta de la tarjeta en el roadmap. |
| `icon` | Ícono de una colección del centro de ayuda (en su _index). |

> [!tip] Plantillas listas
> La carpeta **Templates** de la bóveda tiene plantillas de página de la documentación, elemento del roadmap y versión del changelog. Usa el comando **Plantillas: Insertar plantilla** de Obsidian para empezar una nota con las propiedades correctas.
