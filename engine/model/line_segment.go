package model

/*
#include "LineSegment.hpp"


double LineSegment::slope() const
{
  // simply the rise-over-run calculation
  return (B.y - A.y) / (B.x - A.x);
}

double LineSegment::yIntercept() const     // get B for the line through this segment
{
  // we know the slope, and we know a point.  Easy - b = y-mx
  return (A.y - slope()*A.x);
}


double LineSegment::length() const
{
  // simply the pythagorean hypotenuse
  return sqrt( (B.x-A.x)*(B.x-A.x) + (B.y-A.y)*(B.y-A.y) );
}

bool LineSegment::onLine(Coord C, double tolerance) const
{
  // check coord x against point-slope formula
  // point-slope: Y-y = m(X-x)
  if( fabs( (C.y - A.y) - slope()*(C.x - A.x) ) < tolerance) return true;
  else return false;
}

bool LineSegment::onSegment(Coord C, double tolerance) const
{
  // C must be on the line
  if( !onLine(C, tolerance) ) return false;

  // additionally, C must be between A and B
  if( (A < C) && (C < B) ) return true;
  else return false;
}

LineSegment& LineSegment::operator=(const LineSegment& l)
{
  A = l.A;
  B = l.B;
  _strong = l._strong;

  normalize();

  return *this;
}

bool LineSegment::operator==(const LineSegment& l) const
{
  return ( (A==l.A) && (B==l.B) );
}

bool LineSegment::colinear(const LineSegment& l) const   // lines are colinear if their slopes are the same AND if there is a point they both pass through
{
  if(l.slope() != slope()) return false;

  // assert: lines are the same slope

  LineSegment buffer1(l); // to preserve const
  LineSegment buffer2(*this);

  if( findIntersection(buffer1, buffer2, true) == noCoord) // check for a point lying on both lines
    return false;

  // assert: we found a point that lies on both lines
  return true;
}


// weight coordinate values by length of lines
bool LineSegment::operator<(const LineSegment& l) const
{
  // find a way to compare using both endpoints

  if(l == *this) return false;  // get around problems

  return (A < l.A);
  //double Lx = (l.A.x + l.B.x)*l.length();   // TODO: ask other people about this approach... seems suspicious
  //double Ly = (l.A.y + l.B.y)*l.length();
  //double Tx = (  A.x +   B.x)*length();
  //double Ty = (  A.y +   B.y)*length();

  //if(Ty < Ly) return true;  // first compare on Y
  //if(Tx < Lx) return true;  // then compare on X
  //return false;
}



CPoly LineSegment::asPoly(Color bdyColor)
{
if(B == noCoord) // this is a point, not actually a segment
{
CPoly c;
c.bdy.push_back(A);
c.bdyColor = GREEN;
return c;
}

CPoly c;
c.bdy.push_back(A);
c.bdy.push_back(B);
if(approxEqual(bdyColor, noColor)) { // to return the proper color
if(approxEqual(color, noColor)) c.bdyColor = WHITE;
else c.bdyColor = color;
}

return c;
}


bool LineSegment::isStrong(double thresh, Image* binaryEdgeMap, bool recompute)
{
bool retval = false;

// process arguments, cached status
if(recompute) _strong_cached = false;
if(_strong_cached) return _strong;
if(binaryEdgeMap == NULL) return false;

// assert: binaryEdgeMap is non-null, and we need to recompute from the image


// TODO: insert Ian's isStrongEdge code

_strong = retval; // cache the value
_strong_cached = true;

return retval;
}






Coord findIntersection(LineSegment& one, LineSegment& two, bool extrapolate)
{
Coord foundCoord = noCoord; // return variable, defaults to noCoord if cannot find

// get point-slope information for each line
// point-slope: form Y-y1 = m(X-X1)
one.normalize(); two.normalize();
double m1 = one.slope();
double m2 = two.slope();


Coord pt1 = one.A;
Coord pt2 = two.A;

// check for same-slope ... if so, then must do something else
if(m1 == m2)
{
// check to see if the lines have overlapping endpoints - if so, choose one of those endpoints
if( one.onSegment(two.A) )
foundCoord = two.A;
if( one.onSegment(two.B) )
foundCoord = two.B;
if( two.onSegment(one.A) )
foundCoord = one.A;
if( two.onSegment(one.B) )
foundCoord = one.B;
}
else // if lines have different slopes, derive coordinates of intersection from intersection of point-slope formulas
{
// edge case: check for infinite slope on either one
if(isinf<double>(m1)) {
// TODO
}
if(isinf<double>(m2)) {
// TODO
}


foundCoord.x = (pt2.y - pt1.y + m1*pt1.x - m2*pt2.x) / (m1 - m2);
foundCoord.y = m1*(foundCoord.x - pt1.x) + pt1.y;

// if we are constraining this to the actual Points within the segments, need to make sure it exists on both segments
if(!extrapolate) {
if( !one.onSegment(foundCoord) || !two.onSegment(foundCoord) ) {
foundCoord = noCoord; // cannot find this point on both segments
}
}
}

return foundCoord;
}


Coord findIntersection(LineSegment& one, LineSegment& two, double extrapolatePercentage)
{
// extrapolate both lines out by making new segments that are larger

one.normalize(); two.normalize(); // A < B

// line one
double ext1 = extrapolatePercentage*one.length();
Coord A1 (one.A.x-ext1, one.A.y-ext1*one.slope());
Coord B1 (one.B.x+ext1, one.B.y+ext1*one.slope());
LineSegment L1 (A1, B1);

// line one
double ext2 = extrapolatePercentage*two.length();
Coord A2 (two.A.x-ext2, two.A.y-ext2*two.slope());
Coord B2 (two.B.x+ext2, two.B.y+ext2*two.slope());
LineSegment L2 (A2, B2);

// lines are now ready for intersection check
return findIntersection(L1, L2, false);
}




double angleBetween(const LineSegment& one, const LineSegment& two)  // will return angle in radians measured from line one to line two -- negative if one has greater slope than two
{
// assume both lines pass through the origin -- find the angles each form with the x axis using the slope
// atan returns -pi/2 to +pi/2
double angle1 = atan(one.slope());
double angle2 = atan(two.slope());

double ret = 0.0;

// If the slopes of the lines have the same, they are in the same quadrent
if( (one.slope() >= 0 && two.slope() >= 0) ||
(one.slope() <  0 && two.slope() <  0)) {

// simply return the difference between the angles of the two lines
ret = angle1 - angle2;
}
else {  // lines fall in different quadrents
// Add the absolute values of the two lines, then set the sign based on which slope has the greater absolute value
ret = fabs(angle1) + fabs(angle2);

// if line one has a slope > 0, then sign of difference is negative
if(one.slope() > 0)
{
ret = -1*ret;
}
}


return ret;
}

vector<LineSegment> polyToSegments(const CPoly& poly)
{
vector<LineSegment> ret;

// NOTE: must check with Olaf about edges of this poly... how do we know which Points to connect as edges?
size_t bound = poly.bdy.size()-1;
for(size_t i = 0; i < bound; i++)
{
// first point - at index i
Coord a = poly.bdy[i];

// second point -- next in the vector, or if this is the last element, the first in vector
Coord b;
if(i == bound) {
b = poly.bdy[0];
}
else {
b = poly.bdy[i+1];
}

// make the segment, push it back
LineSegment s(a, b, poly, false);
ret.push_back(s);
}

return ret;
}



int LineSegment::init(const string line)
{
// buffer vars
double Ax, Bx, Ay, By;

// map in all data from string
stringstream ss(line);
string header;
ss >> header; if(ss.fail() || ss.eof()) return -10;
if(header != "DOOR") return -100;
ss >> Ax;     if(ss.fail() || ss.eof()) return -11;
ss >> Ay;     if(ss.fail() || ss.eof()) return -12;
ss >> Bx;     if(ss.fail() || ss.eof()) return -13;
ss >> By;     if(ss.fail()            ) return -14;

// assert: read all successfully -- okay to copy over
_strong = false;  _strong_cached = false;
_parent = noPoly;

A.x = Ax;  A.y = Ay;
B.x = Bx;  B.y = By;

return 0;
}



string LineSegment::line()
{
stringstream ss;

ss << "DOOR " << A.x << " " << A.y << " " << B.x << " " << B.y;

return ss.str();
}


vector<CPoly> asPolys(vector<LineSegment> l)
{
vector<CPoly> ret;
size_t bound = l.size();
for (size_t i = 0; i < bound; i++)
{
ret.push_back( l[i].asPoly() );
}
return ret;
}
*/
