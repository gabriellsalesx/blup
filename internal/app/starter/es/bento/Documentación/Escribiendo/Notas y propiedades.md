---
title: Notas y propiedades
description: Crea una nota y se convierte en página. Las propiedades controlan el resto.
weight: 1
---
Para crear una página, crea una nota en la carpeta correcta. Las **propiedades** (el bloque al inicio de la nota, entre líneas `---`) son opcionales y controlan cómo aparece.

- **En Blup Studio**: haz clic en **Nueva nota** (o `Ctrl+N`) y completa el panel **Propiedades**, encima del texto. Mira [Propiedades de la nota](../Blup%20Studio/Propiedades%20de%20la%20nota.md).
- **En cualquier editor de texto**: escribe `---` en la primera línea, después las propiedades y otra línea `---` para cerrar el bloque.

## En todas las páginas

| Propiedad | Para qué sirve |
| --- | --- |
| `title` | Título de la página. Sin ella, vale el nombre de la nota. |
| `linkTitle` | Título corto para menús y tarjetas. |
| `description` | Resumen en las listas, en Google y en las redes sociales. |
| `weight` | Orden en el menú lateral y en las listas: el menor va primero. |
| `draft` | `true` deja la nota fuera del sitio (aparece solo en la vista previa). |
| `hidden` | `true` la publica, pero la saca de las listas y le pide a Google que no la muestre. |
| `password` | Protege la página con contraseña. Mira [Contraseñas y páginas ocultas](../Configuración%20del%20sitio/Contraseñas%20y%20páginas%20ocultas.md). |

## En los posts del blog y en el changelog

| Propiedad | Para qué sirve |
| --- | --- |
| `date` | Fecha. Sin ella, vale `created`. |
| `updated` | Fecha de la última actualización. |
| `tags` | Temas; cada etiqueta tiene su propia página. |
| `image` | Portada, como `"[[mi-portada.jpg]]"`. |
| `authors` | Quién escribió, como `[Ana, Bruno]`. |
| `featured` | `true` elige el post destacado. |

## En el roadmap y en el centro de ayuda

| Propiedad | Para qué sirve |
| --- | --- |
| `status` | Columna del roadmap: planeado, en progreso o completado. |
| `category` | Etiqueta de la tarjeta en el roadmap. |
| `icon` | Ícono de una colección del centro de ayuda (en su _index). |

## Las que Blup completa solo

| Propiedad | Quién la completa |
| --- | --- |
| `audio`, `audio_hash`, `audio_seconds` | La narración. Mira [Narración de los posts](../Funciones%20con%20IA/Narración%20de%20los%20posts.md). |
| `translationKey`, `translation_hash`, `ai_translated` | La traducción. Mira [Sitios en varios idiomas](../Configuración%20del%20sitio/Sitios%20en%20varios%20idiomas.md). |
| `created`, `updated` | El Studio, en las notas de las bóvedas. |

> [!tip] Plantillas de la línea de comandos
> Los sitios creados desde la línea de comandos reciben la carpeta **Plantillas**, con plantillas de página de la documentación, elemento del roadmap y versión del changelog. Copia una para empezar una nota con las propiedades correctas.
