from filterpy.kalman import KalmanFilter
from filterpy.common import Q_discrete_white_noise

from filterpy.common import Saver
import numpy as np
import pandas as pd

def XYKalman(x, P, R, Q=0.0, dt=1.0):
    """
    Returns a Kalman Filter with

    Args:
        x: inital state vector
        P: covariance of state
        R: measurement noise covariance
        Q: process covariance
        dt: timestep duration
    """

    kf = KalmanFilter(dim_x=4, dim_z=2)
    kf.x = np.array([x[0], x[1], 0, 0])
    kf.F = np.eye(4)
    kf.F[0, 2] = dt
    kf.F[1, 3] = dt

    kf.H = np.eye(2, 4) 
    kf.R = np.eye(2) * R

    if np.isscalar(P):
        kf.P = np.eye(4) * P
    else:
        np.copyto(kf.P, P)
    if np.isscalar(Q):
        kf.Q = Q_discrete_white_noise(dim=4, dt=dt, var=Q)
    else:
        np.copyto(kf.Q, Q)
    return kf

def test(filen='outwalk.csv'):
    kf = XYKalman([0, 0], P=50000, R=64, Q=0.001, dt=0.10)
    s = Saver(kf)
    df = pd.read_csv(filen)
    results = np.zeros((0, 6))
    for _, row in df.iterrows():
        kf.predict()
        inv = np.array([row['x'], row['y']])
        invn = inv + np.random.randn(2) * 8
        kf.update(invn)
        s.save()
        print('Input: {}, Actual: {}, Out: {}'.format(invn, inv, kf.x[0:2]))
        results = np.vstack((results, [inv[0], inv[1], invn[0], invn[1], kf.x[0], kf.x[1]]))
    results = pd.DataFrame(results, columns=['realx', 'realy', 'noisex', 'noisey', 'estx', 'esty'])
    results.to_csv('filtered-out-walk.csv')
    return results
if __name__ == '__main__':
    test()

