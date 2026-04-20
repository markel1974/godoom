#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aTexCoords;
layout (location = 2) in vec3 aOrigin;
layout (location = 3) in float aIsBillboard;
layout (location = 4) in vec3 aPosNext;
layout (location = 5) in float aLerp;
layout (location = 6) in float aYaw;

out vec3 TexCoords;
out float FragDepth;
out vec3 ViewPos;
out vec4 FragPosLightRoom;
out vec4 FragPosLightFlash;

uniform mat4 u_view;
uniform mat4 u_projection;
uniform mat4 u_roomSpaceMatrix;
uniform mat4 u_flashSpaceMatrix;

void main()
{
    TexCoords = aTexCoords;

    vec4 worldPos;

    if (aIsBillboard >= 1.0 && aIsBillboard < 1.5) {
        // --- VIEWPOINT BILLBOARDING (Allineamento al Player) ---

        // 1. Estrazione matematica della Posizione della Camera in World Space
        // Sfruttiamo la matrice inversa della rotazione per trovare le coordinate esatte del giocatore
        vec3 camPos = -transpose(mat3(u_view)) * u_view[3].xyz;

        // 2. Calcoliamo il vettore che "guarda" verso la telecamera
        vec3 toCamera = camPos - aOrigin;
        vec3 right, up;
        if (aIsBillboard > 1.05) {
            // --- BILLBOARD SFERICO (Fumo, Proiettili, Plasma) ---
            // Lo sprite ti guarda dritto negli occhi, da qualsiasi altezza
            if (length(toCamera) < 0.001) {
                toCamera = vec3(0.0, 0.0, 1.0);
            }
            vec3 forward = normalize(toCamera);
            vec3 worldUp = vec3(0.0, 1.0, 0.0);

            // Sicurezza anti-Gimbal Lock (se guardi lo sprite perfettamente dall'alto o dal basso)
            if (abs(forward.y) > 0.999) {
                right = vec3(1.0, 0.0, 0.0);
            } else {
                right = normalize(cross(worldUp, forward));
            }
            up = cross(forward, right);
        } else {
            // --- BILLBOARD CILINDRICO (Nemici, Barili, Alberi) ---
            // Annulliamo l'asse Y: lo sprite ruota solo orizzontalmente e resta piantato a terra
            toCamera.y = 0.0;
            if (length(toCamera) < 0.001) {
                toCamera = vec3(0.0, 0.0, 1.0); // Fallback di sicurezza
            }
            vec3 forward = normalize(toCamera);
            // La "Destra" è ortogonale all'asse Y del mondo e alla direzione verso il player
            right = normalize(cross(vec3(0.0, 1.0, 0.0), forward));
            up = vec3(0.0, 1.0, 0.0);
        }
        // 3. Assembliamo i vertici ignorando aYaw (gli sprite devono solo guardare la camera)
        worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    } else if (aIsBillboard > 1.5) {
        // --- MODELLI 3D (MD2 / MDL) ---
        // 1. Interpolazione Hardware (Costo zero sulla GPU)
        vec3 lPos = mix(aPos, aPosNext, aLerp);
        // 2. Rotazione Yaw (Orizzontale per l'entità nel mondo)
        float cosY = cos(aYaw);
        float sinY = sin(aYaw);
        // Ripristiniamo gli assi originali del MD2
        float origX = lPos.x;
        float origY = -lPos.z;
        // Rotazione 2D orizzontale
        float rotX = (origX * cosY) - (origY * sinY);
        float rotY = (origX * sinY) + (origY * cosY);
        // Mappatura OpenGL
        vec3 rotatedPos = vec3(rotX, lPos.y, -rotY);
        worldPos = vec4(aOrigin + rotatedPos, 1.0);
    } else {
        // --- GEOMETRIA BSP ---
        worldPos = vec4(aPos, 1.0);
    }

    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);

    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;

    gl_Position = u_projection * viewPos;
}