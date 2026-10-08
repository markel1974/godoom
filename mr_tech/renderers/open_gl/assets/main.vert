#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aTexCoords;
layout (location = 2) in vec3 aOrigin;
layout (location = 3) in float aRenderMode;
layout (location = 4) in vec3 aPosNext;
layout (location = 5) in float aLerp;
layout (location = 6) in float aYaw;

out vec3 TexCoords;
out float FragDepth;
out vec3 ViewPos;
out vec4 FragPosLightRoom;
out vec4 FragPosLightFlash;
out float IsFullbright;

uniform mat4 u_view;
uniform mat4 u_projection;
uniform mat4 u_roomSpaceMatrix;
uniform mat4 u_flashSpaceMatrix;
uniform float u_time;

void main() {
    TexCoords = aTexCoords;
    vec4 worldPos;
    if (aRenderMode >= 1.0 && aRenderMode < 1.5) {
        // VIEWPOINT BILLBOARDING (Player Alignment)
        // Mathematical extraction of Camera Position in World Space
        // Leverage the inverse rotation matrix to find the player's exact coordinates
        vec3 camPos = -transpose(mat3(u_view)) * u_view[3].xyz;
        // Calculate vector pointing towards the camera
        vec3 toCamera = camPos - aOrigin;
        vec3 right, up;
        if (aRenderMode > 1.05) {
            // SPHERICAL BILLBOARD (Smoke, Projectiles, Plasma)
            // The sprite faces directly towards the camera from any elevation
            if (length(toCamera) < 0.001) {
                toCamera = vec3(0.0, -0.99, 1.0);
            }
            vec3 forward = normalize(toCamera);
            vec3 worldUp = vec3(0.0, 1.0, 0.0);
            // Anti-Gimbal Lock guard (looking at the sprite directly from above or below)
            if (abs(forward.y) > 0.999) {
                right = vec3(1.0, 0.0, 0.0);
            } else {
                right = normalize(cross(worldUp, forward));
            }
            up = cross(forward, right);
        } else {
            // CYLINDRICAL BILLBOARD (Enemies, Barrels, Trees)
            // Zero out Y axis: sprite rotates horizontally only and remains grounded
            toCamera.y = 0.0;
            if (length(toCamera) < 0.001) {
                toCamera = vec3(0.0, -0.99, 1.0); // Safety fallback
            }
            vec3 forward = normalize(toCamera);
            // Right is orthogonal to world Y axis and direction towards the player
            right = normalize(cross(vec3(0.0, 1.0, 0.0), forward));
            up = vec3(0.0, 1.0, 0.0);
        }
        // Assemble vertices ignoring aYaw (sprites only need to face the camera)
        worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    } else if (aRenderMode > 1.5) {
        // 3D MODELS
        // Hardware Interpolation (Zero GPU overhead)
        vec3 lPos = mix(aPos, aPosNext, aLerp);
        // Yaw Rotation (Horizontal for world entity)
        float cosY = cos(aYaw);
        float sinY = sin(aYaw);
        // Restore MD2 original axes
        float origX = lPos.x;
        float origY = -lPos.z;
        // 2D horizontal rotation
        float rotX = (origX * cosY) - (origY * sinY);
        float rotY = (origX * sinY) + (origY * cosY);
        // OpenGL mapping
        vec3 rotatedPos = vec3(rotX, lPos.y, -rotY);
        worldPos = vec4(aOrigin + rotatedPos, 1.0);
    } else if (aRenderMode > 0.4 && aRenderMode < 0.6) {
        // ANIMATED MATERIALS (e.g., Water, Lava etc)
        const float WATER_WAVE_X_FREQUENCY = 0.05;
        const float WATER_WAVE_Z_FREQUENCY = 0.05;
        const float WATER_WAVE_X_SPEED = 2.0;
        const float WATER_WAVE_Z_SPEED = 1.5;
        const float WATER_WAVE_AMPLITUDE = 2.0;
        const float WATER_UV_SPEED_X = 0.1;
        const float WATER_UV_SPEED_Y = 0.05;
        worldPos = vec4(aPos, 1.0);
        // Generate physical wave motion on vertices
        float waterTime = u_time;
        worldPos.y += sin(worldPos.x * WATER_WAVE_X_FREQUENCY + waterTime * WATER_WAVE_X_SPEED) * WATER_WAVE_AMPLITUDE;
        worldPos.y += cos(worldPos.z * WATER_WAVE_Z_FREQUENCY + waterTime * WATER_WAVE_Z_SPEED) * WATER_WAVE_AMPLITUDE;
        // Scroll UV coordinates to simulate flow
        TexCoords.x += waterTime * WATER_UV_SPEED_X;
        TexCoords.y += waterTime * WATER_UV_SPEED_Y;
    } else if (aRenderMode >= 2.5 && aRenderMode < 3.5) {
        // HUD / SCREEN SPACE (2D)
        // Bypass the world matrix and u_view completely!
        float aspect = 1.777;
        float scale = 0.05; // 10x BIGGER so we can definitely see it!
        float ndcX = (aPos.x * scale) / aspect;
        float ndcY = (aPos.y - 8.0) * scale;
        
        // Z = 0.5 ensures it is perfectly in the middle of the clip volume!
        // This guarantees it won't be culled by the near or far planes.
        gl_Position = vec4(ndcX, ndcY, 0.5, 1.0);
        
        TexCoords = aTexCoords;
        ViewPos = vec3(0.0, 0.0, -1.0); 
        FragDepth = 0.0;
        FragPosLightRoom = vec4(0.0);
        FragPosLightFlash = vec4(0.0);
        IsFullbright = 1.0;
        return;
    } else {
        worldPos = vec4(aPos, 1.0);
    }

    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);

    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
	IsFullbright = (aRenderMode > 0.4 && aRenderMode < 0.6) ? 1.0 : 0.0;

    gl_Position = u_projection * viewPos;
}