#version 330 core
layout (location = 0) out vec4 gPositionDepth;
layout (location = 1) out vec4 gNormal;

in vec3 ViewPos;
in vec2 TexCoords;
in float FragDepth;

uniform sampler2D u_texture;

void main() {
    if(texture(u_texture, TexCoords).a < 0.5) discard;

    gPositionDepth = vec4(ViewPos, FragDepth);

    // Ricalcolo della normale geometrica su GPU (Zero-Math su CPU)
    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);
    vec3 geoNormal = normalize(cross(dp1, dp2));

    // Sicurezza Anti-Backface
    if (geoNormal.z < 0.0) {
        geoNormal = -geoNormal;
    }

    gNormal = vec4(geoNormal, 1.0);
}