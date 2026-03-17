# mr_tech Engine

# mr_tech Engine

A bare-metal, highly decoupled 3D rendering and physics engine written in Go. Designed with a strict system-level architecture, it exposes the raw mechanics of spatial partitioning, portal-based visibility, rigid-body kinematics, and hardware graphics pipelines without relying on high-level commercial abstractions.

## Architecture

The engine operates on a compiler-like pipeline, strictly decoupling geometry ingestion (frontends), logical entities, and the rendering/physics backends via an Intermediate Representation (IR).
### Physics & Spatial Partitioning
The engine discards static blockmaps in favor of a universal, dynamic physical world.
* **Dynamic AABB Tree:** Provides O(log n) spatial queries and collision detection. Every entity (player, NPCs, projectiles) is a first-class node within the tree.
* **Newtonian Kinematics:** Implements real-time rigid-body dynamics featuring momentum conservation, friction, and elastic/inelastic responses.
* **Baumgarte Stabilization:** Resolves inter-frame positional penetration (Positional Projection) ensuring jitter-free collision response.
* **Bidirectional Sync-Back:** The physical solver and logical spatial topology are strictly synchronized. Passive physical deltas (bounces, impacts) are automatically translated back into sector transitions and portal resolution.

### Entity System (Composition over Inheritance)
Entities are designed using idiomatic Go composition, entirely decoupling game logic from the physics and rendering cores.
* **Polymorphic Design:** A central `IThing` interface drives the game loop.
* **Core Encapsulation:** A `ThingBase` struct centralizes the complex math for kinematic vectors and spatial sector synchronization.
* **Behavioral Specialization:** Entities (`ThingEnemy`, `ThingItem`, `ThingProjectile`) embed `ThingBase` to inherit physics and topology natively, overriding only specific routines (like AI tracking or linear projectile motion) without polluting the core engine.

### Frontends (Geometry Ingestion)
Geometry is parsed and translated into a unified planar topology (IR).
* **WAD Parser:** Decodes classic WAD formats to extract raw geometry, completely ignoring legacy BSP trees.
* **Custom DSL:** A text-based parser for rapid, textual planar topology prototyping.
* **Procedural Generator:** An automated dungeon generator (Drunkard's walk) featuring dynamic portal resolution on shared grid edges and continuous elevation handling.

### Rendering Backends
* **Portal Rendering Core:** Dynamic visibility resolution and spatial culling utilizing a custom ring queue and visibility caching.
* **OpenGL Rasterizer:** Hardware-accelerated pipeline featuring geometry batching and custom shader implementations (including SSAO for depth enhancement on planar geometry).
* **Software Rasterizer:** A dependency-free implementation for direct vertex buffer manipulation and raw primitive rasterization.

## Design Philosophy

This project acts as an explicit study of graphics internals, applied mathematics, and system programming. It avoids the "black-box" nature of modern 3D frameworks, offering granular, driver-level control over memory, spatial structures, and the rendering pipeline.
![alt text](https://github.com/markel1974/godoom/blob/main/bloob/screenshot1.png?raw=true)
