import { useEffect } from 'react';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';

export function useWailsEvent(eventName: string, callback: (...args: any[]) => void) {
  useEffect(() => {
    EventsOn(eventName, callback);
    return () => {
      EventsOff(eventName);
    };
  }, [eventName, callback]);
}
