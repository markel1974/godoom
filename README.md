# mr_tech Engine

A bare-metal, highly decoupled 3D rendering engine written in Go. Designed with a system-level architecture, it exposes the raw mechanics of spatial partitioning, portal-based visibility, and hardware graphics pipelines without relying on high-level commercial abstractions.

## Architecture

The engine follows a compiler-like pipeline, strictly separating geometry ingestion (frontends) from spatial resolution and rendering (backends) via an Intermediate Representation (IR).

### Frontends (Geometry Ingestion)
Geometry is parsed and translated into a unified planar topology (IR). Legacy spatial structures (e.g., BSP) are intentionally discarded in favor of pure mesh extraction.
* **WAD Parser:** Decodes classic WAD formats to extract raw geometry and assets.
* **Custom DSL:** A text-based parser for rapid, textual planar topology prototyping.
* **Procedural Generator:** Implements an automated dungeon generator (Drunkard's walk algorithm) with dynamic portal resolution on shared grid edges and continuous elevation handling.

### Core Systems
* **Portal Rendering Engine:** Dynamic visibility resolution and spatial culling utilizing a custom ring queue and visibility caching.
* **Physics & Collision:** Custom AABB tree implementation for high-performance spatial queries and collision detection.

### Rendering Backends
* **OpenGL Rasterizer:** Hardware-accelerated pipeline featuring geometry batching and custom shader implementations (including SSAO).
* **Software Rasterizer:** A raw, dependency-free implementation for direct vertex buffer manipulation and primitive rasterization.

## Design Philosophy

This project acts as an explicit study of graphics internals and system programming. It avoids the "black-box" nature of modern 3D frameworks, offering granular, driver-level control over memory, spatial structures, and the rendering pipeline.


![alt text](https://github.com/markel1974/godoom/blob/main/bloob/screenshot1.png?raw=true)
