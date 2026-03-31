#version 330 core
layout (location = 0) out vec4 gPositionDepth;
layout (location = 1) out vec4 gNormal;

in vec3 ViewPos;
in vec3 TexCoords;
in float FragDepth;
uniform sampler2DArray u_texture;

void main() {
    // Early discard per la trasparenza
    if(texture(u_texture, TexCoords).a < 0.5) discard;

    // Scrive Posizione e Profondità (necessari per SSAO)
    gPositionDepth = vec4(ViewPos, FragDepth);

    // Calcoliamo la variazione di ViewPos rispetto alle coordinate schermo X e Y
    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);

    // Il prodotto vettoriale delle derivate ci dà la normale geometrica perfetta
    vec3 geoNormal = normalize(cross(dp1, dp2));

    // Sicurezza Anti-Backface (Winding safety)
    // In View Space la telecamera guarda verso -Z, una normale che guarda
    // verso la telecamera deve avere Z positiva.
    if (geoNormal.z < 0.0) {
        geoNormal = -geoNormal;
    }

    // Scrive la normale geometrica nel G-Buffer
    gNormal = vec4(geoNormal, 1.0);
}